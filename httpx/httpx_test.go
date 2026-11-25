package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailStatuses(t *testing.T) {
	cases := []struct {
		err  error
		code int
		body string
	}{
		{ErrNotFound, 404, "Not found"},
		{fmt.Errorf("task: %w", ErrNotFound), 404, "Not found"},
		{Invalid("Title is %s", "empty"), 400, "Title is empty"},
		{fmt.Errorf("disk full"), 500, "Internal server error"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		Fail(rec, httptest.NewRequest("GET", "/api/x", nil), c.err)
		if rec.Code != c.code || !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%v: got %d %s", c.err, rec.Code, rec.Body)
		}
	}
}

func TestDecode(t *testing.T) {
	var v struct{ Name string }
	rec := httptest.NewRecorder()
	if !Decode(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"a"}`)), &v) || v.Name != "a" {
		t.Fatalf("valid body: %d %+v", rec.Code, v)
	}
	for body, code := range map[string]int{"": 400, "{": 400, `"` + strings.Repeat("x", MaxJSONBody) + `"`: 413} {
		rec := httptest.NewRecorder()
		if Decode(rec, httptest.NewRequest("POST", "/", strings.NewReader(body)), &v) || rec.Code != code {
			t.Errorf("body of %d bytes: got %d, want %d", len(body), rec.Code, code)
		}
	}
}

func TestRejectCrossSite(t *testing.T) {
	h := RejectCrossSite(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for _, c := range []struct {
		method, site string
		code         int
	}{{"POST", "cross-site", 403}, {"GET", "cross-site", 200}, {"POST", "same-origin", 200}, {"POST", "", 200}} {
		req := httptest.NewRequest(c.method, "/", nil)
		if c.site != "" {
			req.Header.Set("Sec-Fetch-Site", c.site)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.code {
			t.Errorf("%s %q: got %d, want %d", c.method, c.site, rec.Code, c.code)
		}
	}
}
