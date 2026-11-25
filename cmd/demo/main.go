// Command demo serves a page that uses every kit primitive, for the browser
// tests (tests/e2e) and to try a change by hand:
//
//	go run ./cmd/demo -no-auth   # then http://127.0.0.1:8099
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/bhr-tools/kit/auth"
	"github.com/bhr-tools/kit/server"
	"github.com/bhr-tools/kit/store"
)

//go:embed web
var files embed.FS

func main() {
	cfg := server.Flags(8099, "demo.db")
	flag.Parse()
	db, err := store.Open(cfg.DBPath(), nil)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()
	a, err := auth.New(db, "demo_session", cfg.NoAuth)
	if err != nil {
		log.Fatal(err)
	}
	api := http.NewServeMux()
	a.Routes(api)
	web, _ := fs.Sub(files, "web")
	server.Run("Kit demo", cfg.Addr(), server.Handler(server.Options{API: api, Web: web, Ping: db.Ping, Offline: true}))
}
