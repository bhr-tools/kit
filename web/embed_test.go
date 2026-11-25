package web

import (
	"io/fs"
	"testing"
)

func TestBaseCSSEmbedded(t *testing.T) {
	b, err := fs.ReadFile(Files, "css/base.css")
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("css/base.css is empty")
	}
}
