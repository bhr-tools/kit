package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

var app = fstest.MapFS{
	"index.html":    {Data: []byte("<!doctype html><title>App</title>")},
	"css/style.css": {Data: []byte(`@import url("/kit/css/kit.css");`)},
}

func TestRoutes(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("api")) })
	h := Handler(Options{API: api, Web: app, Offline: true})

	if rec := get(t, h, "/"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>App") {
		t.Errorf("/: %d %s", rec.Code, rec.Body)
	}
	if rec := get(t, h, "/api/tasks"); rec.Body.String() != "api" {
		t.Errorf("/api/: %s", rec.Body)
	}
	if rec := get(t, h, "/kit/css/base.css"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "--accent") {
		t.Errorf("/kit/: %d", rec.Code)
	}
	if rec := get(t, h, "/kit/js/overlay.js"); rec.Code != 200 || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("/kit/js: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	rec := get(t, h, "/healthz")
	if rec.Code != 200 || rec.Header().Get("Content-Security-Policy") != DefaultCSP {
		t.Errorf("/healthz: %d, CSP %q", rec.Code, rec.Header().Get("Content-Security-Policy"))
	}
}

func TestServiceWorkerListsAppAndKitFiles(t *testing.T) {
	sw := get(t, Handler(Options{Web: app, Offline: true}), "/sw.js").Body.String()
	for _, want := range []string{`const VERSION = "`, `"/"`, `"/css/style.css"`, `"/kit/css/base.css"`, `"/kit/js/overlay.js"`, "caches.open"} {
		if !strings.Contains(sw, want) {
			t.Errorf("sw.js lacks %s", want)
		}
	}
	if strings.Contains(sw, `"/kit/sw.js"`) || strings.Contains(sw, ".go\"") {
		t.Error("sw.js lists itself or Go files")
	}
	if rec := get(t, Handler(Options{Web: app}), "/sw.js"); strings.Contains(rec.Body.String(), "VERSION") {
		t.Error("sw.js served without Offline")
	}
}

func TestHealthzFailsWithTheDatabase(t *testing.T) {
	h := Handler(Options{Ping: func() error { return errors.New("closed") }})
	if rec := get(t, h, "/healthz"); rec.Code != 503 {
		t.Errorf("got %d", rec.Code)
	}
}

func TestCustomCSP(t *testing.T) {
	h := Handler(Options{CSP: "default-src 'none'"})
	if got := get(t, h, "/healthz").Header().Get("Content-Security-Policy"); got != "default-src 'none'" {
		t.Errorf("got %q", got)
	}
}

func TestDBPath(t *testing.T) {
	c := &Config{DataDir: "/var/lib/x", dbFile: "x.db"}
	if c.DBPath() != "/var/lib/x/x.db" {
		t.Error(c.DBPath())
	}
	c.DB = "/tmp/y.db"
	if c.DBPath() != "/tmp/y.db" {
		t.Error(c.DBPath())
	}
}
