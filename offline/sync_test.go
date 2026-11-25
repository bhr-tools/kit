package offline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bhr-tools/kit/httpx"
	"github.com/bhr-tools/kit/store"
)

func setup(t *testing.T) (http.Handler, *[]string) {
	db, err := store.Open(filepath.Join(t.TempDir(), "app.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	items := &[]string{}
	s, err := New(db,
		func() (any, error) { return map[string]any{"items": *items}, nil },
		func(op Op) error {
			var x struct{ Name string }
			if err := op.Decode(&x); err != nil {
				return err
			}
			switch {
			case op.Type == "item.missing":
				return httpx.ErrNotFound
			case x.Name == "":
				return httpx.Invalid("A name is needed")
			}
			*items = append(*items, x.Name)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	s.Routes(mux, func(h http.Handler) http.Handler { return h })
	return mux, items
}

func post(h http.Handler, body string) (*httptest.ResponseRecorder, response) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/sync", strings.NewReader(body)))
	var res response
	json.Unmarshal(rec.Body.Bytes(), &res)
	return rec, res
}

const id1 = "11111111-1111-4111-8111-111111111111"
const id2 = "22222222-2222-4222-8222-222222222222"
const id3 = "33333333-3333-4333-8333-333333333333"

func TestBatchAppliedOnce(t *testing.T) {
	h, items := setup(t)
	batch := `{"ops":[
		{"id":"` + id1 + `","type":"item.add","args":{"name":"milk"}},
		{"id":"` + id2 + `","type":"item.add","args":{}},
		{"id":"` + id3 + `","type":"item.missing","args":{"name":"x"}},
		{"id":"nope","type":"item.add","args":{"name":"y"}}]}`
	rec, res := post(h, batch)
	if rec.Code != 200 || len(res.Results) != 4 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	got := []string{res.Results[0].Status, res.Results[1].Status, res.Results[2].Status, res.Results[3].Status}
	if strings.Join(got, ",") != "ok,invalid,not_found,invalid" || res.Results[1].Error != "A name is needed" {
		t.Fatalf("results: %+v", res.Results)
	}
	_, again := post(h, batch)
	if len(*items) != 1 || again.Results[1].Error != "A name is needed" {
		t.Fatalf("replayed: items %v, results %+v", *items, again.Results)
	}
	if !strings.Contains(string(again.Snapshot), `"milk"`) {
		t.Fatalf("snapshot: %s", again.Snapshot)
	}
}

func TestETag(t *testing.T) {
	h, _ := setup(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/sync", nil))
	etag := rec.Header().Get("ETag")
	req := httptest.NewRequest("GET", "/api/sync", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if etag == "" || rec.Code != http.StatusNotModified {
		t.Fatalf("etag %q, code %d", etag, rec.Code)
	}
}

func TestTooManyOps(t *testing.T) {
	h, _ := setup(t)
	ops := make([]string, MaxOps+1)
	for i := range ops {
		ops[i] = `{"id":"x","type":"t"}`
	}
	if rec, _ := post(h, `{"ops":[`+strings.Join(ops, ",")+`]}`); rec.Code != 400 {
		t.Fatalf("got %d", rec.Code)
	}
}
