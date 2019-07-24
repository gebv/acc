package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gebv/acca/api"
	_ "github.com/gebv/acca/engine/strategies/isimple"
	_ "github.com/gebv/acca/engine/strategies/tsberbank"
	_ "github.com/gebv/acca/engine/strategies/tsimple"
	"github.com/gebv/acca/engine/worker"
	"github.com/gebv/acca/interceptors/auth"
	"github.com/gebv/acca/interceptors/recover"
	settingsInterceptor "github.com/gebv/acca/interceptors/settings"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/services"
	"github.com/gebv/acca/services/accounts"
	"github.com/gebv/acca/services/auditor"
	"github.com/gebv/acca/services/invoices"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/labstack/echo"
	echo_middleware "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/nats-io/gnatsd.v2/server"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

var (
	VERSION         = "dev"
	pgConnF         = flag.String("pg-conn", "postgres://acca:acca@127.0.0.1:5432/acca?sslmode=disable", "PostgreSQL connection string.")
	grpcAddrsF      = flag.String("grpc-addrs", "127.0.0.1:10001", "gRPC listen address.")
	grpcReflectionF = flag.Bool("grpc-reflection", false, "Enable gRPC reflection.")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	defaultLogger("INFO")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	zap.L().Info("Starting...", zap.String("version", VERSION))
	defer func() { zap.L().Info("Done.") }()

	syncLogger := developLogger(false)
	defer syncLogger()
	handleTerm(cancel)

	sqlDB := setupPostgres(*pgConnF, 0, 5, 5)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(zap.L().Sugar().Debugf))
	_, err := db.Exec("SELECT version();")
	if err != nil {
		zap.L().Panic("Failed to check version to PostgreSQL.", zap.Error(err))
	}

	sNats, err := server.NewServer(&server.Options{
		Host:           "127.0.0.1",
		Port:           4222,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 2048,
	})
	if err != nil || sNats == nil {
		panic(fmt.Sprintf("No NATS Server object returned: %v", err))
	}

	// Run server in Go routine.
	go sNats.Start()

	// Wait for accept loop(s) to be started
	if !sNats.ReadyForConnections(10 * time.Second) {
		panic("Unable to start NATS Server in Go Routine")
	}

	n, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	nc, err := nats.NewEncodedConn(n, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}

	sberProvider := sberbank.NewProvider(db, sberbank.Config{
		EntrypointURL: os.Getenv("SBERBANK_ENTRYPOINT_URL"),
		Token:         os.Getenv("SBERBANK_TOKEN"),
		Password:      os.Getenv("SBERBANK_PASSWORD"),
		UserName:      os.Getenv("SBERBANK_USER_NAME"),
	},
		nc,
	)

	worker.SubToNATS(nc, db, sberProvider)

	lis, err := net.Listen("tcp", *grpcAddrsF)
	if err != nil {
		zap.L().Panic("Failed to listen.", zap.Error(err))
	}

	// аудитор http запросов (сохраняет в БД все реквесты и респонсы)
	httpAuditor := auditor.NewHttpAuditor(sqlDB)
	defer httpAuditor.Stop()
	prometheus.MustRegister(httpAuditor)

	serv := services.NewService(db)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
			settingsInterceptor.Unary(VERSION),
			recover.Unary(),
			auth.Unary(serv, httpAuditor),
		)),
		grpc.StreamInterceptor(middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
			settingsInterceptor.Stream(VERSION),
			recover.Stream(),
			auth.Stream(serv),
		)),
	)

	accServ := accounts.NewServer(db)
	invServ := invoices.NewServer(db, nc)

	api.RegisterAccountsServer(s, accServ)
	api.RegisterInvoicesServer(s, invServ)

	// graceful stop
	go func() {
		<-ctx.Done()
		nc.Drain()
		n.Drain()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		go func() {
			<-ctx.Done()
			s.Stop()

		}()
		s.GracefulStop()
		sNats.Shutdown()
	}()

	// TODO: Registry servers

	if *grpcReflectionF {
		// for debug via grpcurl
		reflection.Register(s)
	}

	var wg sync.WaitGroup

	zap.L().Info("gRPC server listen address.", zap.String("address", lis.Addr().String()))
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.Serve(lis); err != nil {
			zap.L().Panic("Failed to serve.", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		serverWebhookSberbank(ctx, sberProvider)
	}()

	wg.Wait()

	// - внутренний grpc АПИ
	// - хандлер для платежек

	/*
		Входящая операция падает в общую очередь
		Колторая обрабатывается в горутине
		Все состояния персистятся в PG
		В случае падения процесса очередь воссоздается из БД (то есть сохранять состояния команд?)
	*/
}

// Configure configure zap logger.
//
// Available values of level:
// - DEBUG
// - INFO
// - WARN
// - ERROR
// - DPANIC
// - PANIC
// - FATAL
func defaultLogger(levelSet string) {
	level := zapcore.InfoLevel
	if err := level.Set(levelSet); err != nil {
		panic(err)
	}
	config := zap.NewDevelopmentConfig()
	config.Level.SetLevel(level)
	l, err := config.Build(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)
	zap.RedirectStdLog(l.Named("stdlog"))
}

func developLogger(debug bool) func() error {
	zap.L().Sync()

	var config zap.Config
	config = zap.NewDevelopmentConfig()
	config.Development = true
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	if debug {
		config.Level.SetLevel(zap.DebugLevel)
	} else {
		config.Level.SetLevel(zap.InfoLevel)
	}

	l, err := config.Build(
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(l)
	zap.RedirectStdLog(l.Named("stdlog"))

	return l.Sync
}

func productionLogger(debug bool) func() error {
	zap.L().Sync()

	var config zap.Config
	config = zap.NewProductionConfig()
	config.Development = false
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if debug {
		config.Level.SetLevel(zap.DebugLevel)
	} else {
		config.Level.SetLevel(zap.InfoLevel)
	}

	l, err := config.Build(
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(l)
	zap.RedirectStdLog(l.Named("stdlog"))

	return l.Sync
}

func handleTerm(cancel context.CancelFunc) {
	// handle termination signals: first one gracefully, force exit on the second one
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, unix.SIGTERM, unix.SIGINT)
	go func() {
		s := <-signals
		zap.L().Warn("Shutting down.", zap.String("signal", unix.SignalName(s.(unix.Signal))))
		cancel()

		s = <-signals
		zap.L().Panic("Exiting!", zap.String("signal", unix.SignalName(s.(unix.Signal))))
	}()
}

func setupPostgres(conn string, maxLifetime time.Duration, maxOpen, maxIdle int) *sql.DB {
	sqlDB, err := sql.Open("postgres", conn)
	if err != nil {
		zap.L().Panic("Failed to connect to PostgreSQL.", zap.Error(err))
	}
	sqlDB.SetConnMaxLifetime(maxLifetime)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	if err = sqlDB.Ping(); err != nil {
		zap.L().Panic("Failed to connect ping PostgreSQL.", zap.Error(err))
	}
	zap.L().Info("Postgres - Connected!")

	return sqlDB
}

func serverWebhookSberbank(ctx context.Context, provider *sberbank.Provider) {

	e := echo.New()

	e.Use(echo_middleware.CORSWithConfig(echo_middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET,
			echo.PUT,
			echo.POST,
			echo.DELETE,
			echo.OPTIONS,
			echo.CONNECT,
			echo.HEAD,
			echo.TRACE,
		},
	}))

	e.Use(echo_middleware.Recover())

	e.Use(echo_middleware.Logger())

	e.GET("/webhook/sberbank", provider.SberbankWebhookHandler())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		zap.L().Info("start server sberbank webhook ", zap.String("address", "/webhook/sberbank"))
		if err := e.Start(":10003"); err != nil {
			zap.L().Error("failed run server sberbank webhook", zap.Error(err))
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		Ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := e.Shutdown(Ctx)
		if err != nil {
			zap.L().Error("failed shutdown server sberbank webhook", zap.Error(err))
		}
		err = e.Close()
		if err != nil {
			zap.L().Error("failed close server sberbank webhook", zap.Error(err))
		}
		zap.L().Debug("success shutdown server sberbank webhook")
	}()
	wg.Wait()
}
