package tests

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

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

func runPostgresMigrations() {

	psql, err := exec.LookPath("psql")
	if err != nil {
		log.Panic(err)
	}

	const (
		database = "acca"
		username = "acca"
		password = "acca"
	)

	// wait for PostgreSQL to start
	for i := 0; i < 5; i++ {
		cmd := exec.Command(psql, "-q", "-h", *DockerHostF, "-p", "5433", "-d", database, "-U", username, "-c", "SELECT version();")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
		log.Print(strings.Join(cmd.Args, " "))
		if err = cmd.Run(); err != nil {
			log.Print(err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	files, err := filepath.Glob(filepath.Join("..", "postgres_schema", "*.sql"))
	if err != nil {
		log.Panic(err)
	}

	for _, file := range files {
		cmd := exec.Command(psql, "-q", "-h", *DockerHostF, "-p", "5433", "-d", database, "-U", username, "-v", "ON_ERROR_STOP=1", "-f", file)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
		log.Print(strings.Join(cmd.Args, " "))
		if err = cmd.Run(); err != nil {
			log.Panic(err)
		}
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

func createAccessToken() string {
	out, err := exec.Command(filepath.Join("..", "bin", "acca-race"), "--gen-access-token").Output()
	if err != nil {
		panic(err)
	}
	var conf map[string]interface{}
	if err := json.Unmarshal(out, &conf); err != nil {
		panic(err)
	}
	return conf["access_token"].(string)
}

func setup() {

	runPostgresMigrations()

	runMake("build-race")
}

var DockerHostF = flag.String("docker-host-address", "127.0.0.1", "Docker address host.")

func TestMain(m *testing.M) {
	onlySetupF := flag.Bool("only-setup", false, "Only setup: put settings to Consul, migrate database and exit.")
	skipSetupF := flag.Bool("skip-setup", false, "Skip setup: run tests and exit.")
	grpcAddrF := flag.String("grpc-addr", "127.0.0.1:10011", "gRPC client API address")

	log.SetPrefix("testmain: ")
	log.SetFlags(0)

	flag.Parse()
	if testing.Short() {
		log.Print("-short flag is passed, skipping integration tests.")
		os.Exit(0)
	}

	if *DockerHostF == "" {
		// TODO: есть ли более красивый способ?
		*DockerHostF = "127.0.0.1"
	}

	if *onlySetupF && *skipSetupF {
		log.Fatal("Both -only-setup and -skip-setup are given.")
	}

	var cancel context.CancelFunc
	Ctx, cancel = context.WithCancel(context.Background())
	Ctx = metadata.NewOutgoingContext(Ctx, metadata.Pairs("x-request-id", fmt.Sprintf("e2e-test-%d", time.Now().UnixNano())))

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

	if *onlySetupF {
		setup()
		os.Exit(0)
	}

	var exitCode int
	defer func() {
		if p := recover(); p != nil {
			panic(p)
		}
		log.Printf("Stoped main_test with exit code %d, sleep 10 sec\n", exitCode)
		time.Sleep(10 * time.Second)
		os.Exit(exitCode)
	}()

	if !*skipSetupF {
		runMake("up")
		// defer runMake("down")

		setup()

		AccessToken = createAccessToken()

		go runGo(
			"acca-race",
			"--grpc-reflection=true",
		)
	}

	var err error
	Conn, err = grpc.Dial(*grpcAddrF, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panic(err)
	}

	go func() {
		<-time.After(30 * time.Minute)
		cancel()
	}()
	exitCode = m.Run()
	cancel()
	log.Println("TestMain: bye bye.")
}
