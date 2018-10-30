package tests

import (
	_ "github.com/lib/pq" // register database driver

	"database/sql"
	"flag"
	"log"
	"testing"
)

var db *sql.DB
var accaGrpcAddr string

func TestMain(m *testing.M) {
	flag.StringVar(&accaGrpcAddr, "grpc-addr", "127.0.0.1:3030", "gRPC acca API address")

	log.SetPrefix("testmain: ")
	log.SetFlags(0)

	flag.Parse()

	var err error
	db, err = sql.Open("postgres", "postgres://acca:acca@127.0.0.1:5432/acca?sslmode=disable")
	if err != nil {
		log.Panic("Failed to connect to Postgres:", err)
	}
	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	if err = db.Ping(); err != nil {
		log.Panic("Failed to connect ping Postgres:", err)
	}

	m.Run()
}
