package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/bhr-tools/kit/httpx"
)

func TestMigrationsRunOnceInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "app.db")
	m := []string{`CREATE TABLE a (x INTEGER)`, `INSERT INTO a VALUES (1)`}
	db, err := Open(path, m)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	m = append(m, `INSERT INTO a VALUES (2)`)
	db, err = Open(path, m)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n, version int
	db.QueryRow(`SELECT COUNT(*) FROM a`).Scan(&n)
	db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if n != 2 || version != 3 {
		t.Fatalf("rows %d, version %d", n, version)
	}
}

func TestFailedMigrationIsRolledBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	if _, err := Open(path, []string{`CREATE TABLE a (x)`, `INSERT INTO a VALUES (1); SELECT nope FROM missing`}); err == nil {
		t.Fatal("want an error")
	}
	db, err := Open(path, []string{`CREATE TABLE a (x)`})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM a`).Scan(&n)
	if n != 0 {
		t.Fatalf("the failed migration left %d rows", n)
	}
}

func TestNewerDatabaseIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	db, _ := Open(path, []string{`SELECT 1`, `SELECT 1`})
	db.Close()
	if _, err := Open(path, []string{`SELECT 1`}); err == nil {
		t.Fatal("want an error")
	}
}

func TestClientID(t *testing.T) {
	if id, err := ClientID(""); err != nil || len(id) != 36 {
		t.Errorf("empty: %q %v", id, err)
	}
	if id, _ := ClientID(" 11111111-1111-4111-8111-111111111111 "); id != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("uuid: %q", id)
	}
	var invalid *httpx.ValidationError
	if _, err := ClientID("x"); !errors.As(err, &invalid) {
		t.Errorf("x: %v", err)
	}
}
