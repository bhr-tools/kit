package auth

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bhr-tools/kit/store"
)

func setup(t *testing.T, noAuth bool) (*Auth, http.Handler) {
	t.Helper()
	FailedLoginDelay = 0
	db, err := store.Open(filepath.Join(t.TempDir(), "app.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a, err := New(db, "app_session", noAuth)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	a.Routes(mux)
	mux.Handle("GET /api/data", a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("data")) })))
	return a, mux
}

func do(h http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == "app_session" && c.Value != "" {
			return c
		}
	}
	return nil
}

func TestFlow(t *testing.T) {
	_, h := setup(t, false)

	// Open until a password is set.
	if rec := do(h, "GET", "/api/data", "", nil); rec.Code != 200 {
		t.Fatalf("before setup: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/setup", `{"password":"abc"}`, nil); rec.Code != 400 {
		t.Fatalf("short password: %d", rec.Code)
	}
	rec := do(h, "POST", "/api/auth/setup", `{"password":"secret"}`, nil)
	first := sessionCookie(rec)
	if rec.Code != 200 || first == nil {
		t.Fatalf("setup: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/setup", `{"password":"other1"}`, nil); rec.Code != 400 {
		t.Fatalf("second setup: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/data", "", nil); rec.Code != 401 {
		t.Fatalf("without session: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/data", "", first); rec.Code != 200 {
		t.Fatalf("with session: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/login", `{"password":"wrong"}`, nil); rec.Code != 401 {
		t.Fatalf("wrong password: %d", rec.Code)
	}
	second := sessionCookie(do(h, "POST", "/api/auth/login", `{"password":"secret"}`, nil))
	if second == nil {
		t.Fatal("login gave no session")
	}

	// Changing the password signs out the other sessions, not this one.
	rec = do(h, "POST", "/api/auth/change-password", `{"old_password":"secret","new_password":"better"}`, first)
	kept := sessionCookie(rec)
	if rec.Code != 200 || kept == nil {
		t.Fatalf("change password: %d", rec.Code)
	}
	if do(h, "GET", "/api/data", "", second).Code != 401 || do(h, "GET", "/api/data", "", kept).Code != 200 {
		t.Fatal("sessions after change")
	}
	do(h, "POST", "/api/auth/logout", "", kept)
	if do(h, "GET", "/api/data", "", kept).Code != 401 {
		t.Fatal("session survived logout")
	}
	if !strings.Contains(do(h, "GET", "/api/auth/status", "", nil).Body.String(), `"has_password":true`) {
		t.Fatal("status")
	}
}

func TestNoAuth(t *testing.T) {
	_, h := setup(t, true)
	if !strings.Contains(do(h, "GET", "/api/auth/status", "", nil).Body.String(), `"no_auth":true`) {
		t.Fatal("status")
	}
	if do(h, "GET", "/api/data", "", nil).Code != 200 {
		t.Fatal("data")
	}
}

func TestOutdatedHashIsUpgraded(t *testing.T) {
	a, _ := setup(t, false)
	// GetDone's first release format.
	salt := []byte("0123456789abcdef")
	legacy := strings.ToLower(hexs(salt) + "$" + hexs(legacyKey("secret", salt)))
	a.db.Exec(`INSERT INTO auth_settings (key, value) VALUES ('password_hash', ?)`, legacy)
	if ok, _ := a.VerifyPassword("secret"); !ok {
		t.Fatal("legacy hash refused")
	}
	var stored string
	a.db.QueryRow(`SELECT value FROM auth_settings WHERE key = 'password_hash'`).Scan(&stored)
	if !strings.HasPrefix(stored, "v2$600000$") {
		t.Fatalf("not upgraded: %s", stored)
	}
	if ok, _ := a.VerifyPassword("secret"); !ok {
		t.Fatal("upgraded hash refused")
	}
}

func TestFocusFirstHashIsUpgraded(t *testing.T) {
	a, _ := setup(t, false)
	salt := []byte("0123456789abcdef")
	sum := sha256.Sum256(append(append([]byte{}, salt...), "secret"...))
	a.db.Exec(`INSERT INTO auth_settings (key, value) VALUES ('password_hash', ?)`, "s1$"+hexs(salt)+"$"+hexs(sum[:]))
	if ok, _ := a.VerifyPassword("wrong"); ok {
		t.Fatal("wrong password accepted")
	}
	if ok, _ := a.VerifyPassword("secret"); !ok {
		t.Fatal("Focus hash refused")
	}
	var stored string
	a.db.QueryRow(`SELECT value FROM auth_settings WHERE key = 'password_hash'`).Scan(&stored)
	if !strings.HasPrefix(stored, "v2$600000$") {
		t.Fatalf("not upgraded: %s", stored)
	}
}

func TestAppTokens(t *testing.T) {
	a, _ := setup(t, false)
	token, err := a.CreateAppToken("QuickShell", "timer")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	got := a.RequestAppToken(req)
	if got == nil || got.Name != "QuickShell" || got.Scope != "timer" {
		t.Fatalf("token: %+v", got)
	}
	req.Header.Set("Authorization", "Bearer nope")
	if a.RequestAppToken(req) != nil {
		t.Fatal("unknown token accepted")
	}
	list, _ := a.AppTokens()
	if len(list) != 1 || list[0].UsedAt == nil {
		t.Fatalf("list: %+v", list)
	}
	if err := a.DeleteAppToken(list[0].ID); err != nil {
		t.Fatal(err)
	}
	if a.DeleteAppToken(list[0].ID) == nil {
		t.Fatal("deleted twice")
	}
}

func hexs(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&15])
	}
	return string(out)
}

func TestLinkSecretIsStable(t *testing.T) {
	a, _ := setup(t, false)
	one, err := a.LinkSecret()
	two, _ := LinkSecret(a.db)
	if err != nil || len(one) != 32 || string(one) != string(two) {
		t.Fatalf("%x %x %v", one, two, err)
	}
}

func TestPasswordStampChangesWithThePassword(t *testing.T) {
	a, _ := setup(t, false)
	if s, err := a.PasswordStamp(); err != nil || s != "" {
		t.Fatalf("no password: %q %v", s, err)
	}
	if err := a.SetInitialPassword("first"); err != nil {
		t.Fatal(err)
	}
	first, _ := a.PasswordStamp()
	if err := a.SetPassword("second"); err != nil {
		t.Fatal(err)
	}
	second, _ := a.PasswordStamp()
	if first == "" || first == second {
		t.Fatalf("stamps %q %q", first, second)
	}
}
