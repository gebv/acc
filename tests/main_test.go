package tests

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq" // register database driver
	"golang.org/x/sys/unix"

	"database/sql"
	"flag"
	"log"
	"testing"
)

var (
	db           *sql.DB
	accaGrpcAddr string
)

func TestMain(m *testing.M) {
	flag.StringVar(&accaGrpcAddr, "grpc-addr", "127.0.0.1:3030", "gRPC acca API address")

	log.SetPrefix("testmain: ")
	log.SetFlags(0)

	flag.Parse()

	var cancel context.CancelFunc
	Ctx, cancel = context.WithCancel(context.Background())

	// handle termination signals: first one cancels context, force exit on the second one
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		log.Printf("Got %s, shutting down...", unix.SignalName(s.(unix.Signal)))
		cancel()

		s = <-signals
		log.Panicf("Got %s, exiting!", unix.SignalName(s.(unix.Signal)))
	}()

	var exitCode int
	defer func() {
		if p := recover(); p != nil {
			panic(p)
		}
		os.Exit(exitCode)
	}()

	setupPostgres()

	runMake("build-race")
	go runGo("acca", "--grpc-addr=127.0.0.1:3031")

	time.Sleep(time.Second)

	exitCode = m.Run()
	cancel()
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

func runMake(arg string) {
	args := []string{"-C", "..", arg}
	cmd := exec.Command("make", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Run(); err != nil {
		log.Panic(err)
	}
}

func runGo(bin string, args ...string) {
	cmd := exec.Command(filepath.Join("..", "bin", bin), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), `GORACE="halt_on_error=1"`)
	log.Print(strings.Join(cmd.Args, " "))
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	go func() {
		<-Ctx.Done()
		_ = cmd.Process.Signal(unix.SIGTERM)
	}()

	if err := cmd.Wait(); err != nil {
		log.Print(err)
	}
}
