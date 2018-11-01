package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/gebv/acca/services/accounts"
	"github.com/gebv/acca/services/transfer"

	"github.com/gebv/acca/api/acca"
	"google.golang.org/grpc"
)

var VERSION = "dev"

var (
	listGrpcAddrF = flag.String("grpc-addr", "127.0.0.1:3030", "gRPC server address.")

	db *sql.DB
)

func main() {
	rand.Seed(time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis, err := net.Listen("tcp", *listGrpcAddrF)
	if err != nil {
		log.Panic(err, "Failed to listen.")
	}

	setupPostgres()

	s := grpc.NewServer()

	// graceful stop takes up to stopTimeout
	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		go func() {
			<-ctx.Done()
			s.Stop()
		}()

		s.GracefulStop()
	}()

	accountsServer := accounts.NewServer(db)
	transferServer := transfer.NewServer(db)

	acca.RegisterAccountsServer(s, accountsServer)
	acca.RegisterTransferServer(s, transferServer)

	if err := s.Serve(lis); err != nil {
		log.Panic(err)
	}
}

func setupPostgres() {
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
}
