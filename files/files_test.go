package files

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bhr-tools/kit/httpx"
)

var png = []byte("\x89PNG\r\n\x1a\n0000IHDR")

func newStore(t *testing.T) *Store {
	return New(t.TempDir(), 1<<10, func() ([]byte, error) { return []byte("key"), nil })
}

func TestSaveLimits(t *testing.T) {
	s := newStore(t)
	if n, err := s.Save("a", bytes.NewReader(png)); err != nil || n != int64(len(png)) {
		t.Fatalf("save: %d %v", n, err)
	}
	var invalid *httpx.ValidationError
	if _, err := s.Save("b", strings.NewReader("")); !errors.As(err, &invalid) {
		t.Errorf("empty: %v", err)
	}
	if _, err := s.Save("c", bytes.NewReader(make([]byte, 2<<10))); !errors.As(err, &invalid) {
		t.Errorf("too large: %v", err)
	}
	if _, err := s.Open("c"); !errors.Is(err, httpx.ErrNotFound) {
		t.Errorf("a refused file was kept: %v", err)
	}
}

func TestContentTypeIsSniffed(t *testing.T) {
	cases := map[string]string{"photo.png": "image/png", "notes.txt": "text/plain", "IMG.HEIC": "image/heic", "page.html": "application/octet-stream"}
	heads := map[string][]byte{"photo.png": png, "notes.txt": []byte("hello"), "IMG.HEIC": []byte("\x00\x00\x00 ftypheic"), "page.html": []byte("<html><script>")}
	for name, want := range cases {
		if got := ContentType(heads[name], name); got != want {
			t.Errorf("%s: %s, want %s", name, got, want)
		}
	}
	if CleanFilename(`C:\dir\a`+"\x01"+`.pdf`) != "a.pdf" || CleanFilename("  ") != "file" {
		t.Error("CleanFilename")
	}
}

func TestSignedLinks(t *testing.T) {
	s := newStore(t)
	link := s.Link(LinkPath("/api/files/", "x y"), "x y")
	if !strings.HasPrefix(link, "/api/files/x%20y?exp=") {
		t.Fatal(link)
	}
	req := httptest.NewRequest("GET", link, nil)
	if !s.ValidLink(req, "x y") || s.ValidLink(req, "other") {
		t.Error("signature check")
	}
	old := httptest.NewRequest("GET", "/?exp="+itoa(time.Now().Add(-time.Hour).Unix())+"&sig="+sign([]byte("key"), "x y", itoa(time.Now().Add(-time.Hour).Unix())), nil)
	if s.ValidLink(old, "x y") {
		t.Error("expired link accepted")
	}
}

func TestUploadAndServe(t *testing.T) {
	s := newStore(t)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "../photo.png")
	fw.Write(png)
	mw.Close()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	var got []Upload
	ok := s.ReadUploads(httptest.NewRecorder(), req, 4, func(u Upload) error {
		_, err := s.Save("f1", u.Body)
		got = append(got, u)
		return err
	})
	if !ok || len(got) != 1 || got[0].Filename != "photo.png" || got[0].ContentType != "image/png" {
		t.Fatalf("upload: %v %+v", ok, got)
	}
	rec := httptest.NewRecorder()
	s.Serve(rec, httptest.NewRequest("GET", "/", nil), "f1", "photo.png", "image/png", time.Now())
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), png) || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") ||
		!strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline") {
		t.Fatalf("serve: %d %v", rec.Code, rec.Header())
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
