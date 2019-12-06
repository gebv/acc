package main

import (
	"context"
	"database/sql"
	"flag"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"contrib.go.opencensus.io/exporter/stackdriver"
	firebase "firebase.google.com/go"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/gebv/acca/api"
	_ "github.com/gebv/acca/engine/strategies/invoices/refund"
	_ "github.com/gebv/acca/engine/strategies/invoices/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/moedelo"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank_refund"
	_ "github.com/gebv/acca/engine/strategies/transactions/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe_refund"
	"github.com/gebv/acca/httputils"
	"github.com/gebv/acca/interceptors/auth"
	"github.com/gebv/acca/interceptors/recover"
	settingsInterceptor "github.com/gebv/acca/interceptors/settings"
	"github.com/gebv/acca/services"
	"github.com/gebv/acca/services/accounts"
	"github.com/gebv/acca/services/auditor"
	"github.com/gebv/acca/services/invoices"
	"github.com/gebv/acca/services/updater"
)

var (
	VERSION = "dev"

	onLoggerDev         = flag.Bool("logger-dev", false, "Enable development logger.")
	onLoggerDebugLevelF = flag.Bool("logger-debug-level", false, "Enable debug level logger.")
	grpcReflectionF     = flag.Bool("grpc-reflection", false, "Enable gRPC reflection.")
)

func main() {
	var wg sync.WaitGroup
	rand.Seed(time.Now().UnixNano())
	defaultLogger("INFO")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	zap.L().Info("Starting billing service...",
		zap.String("version", VERSION),
		zap.String("env.GOOGLE_CLOUD_PROJECT", os.Getenv("GOOGLE_CLOUD_PROJECT")),
		zap.String("env.GAE_VERSION", os.Getenv("GAE_VERSION")),
		zap.String("env.GAE_SERVICE", os.Getenv("GAE_SERVICE")),
		zap.String("env.GAE_ENV", os.Getenv("GAE_ENV")),
		zap.String("env.GAE_MEMORY_MB", os.Getenv("GAE_MEMORY_MB")),
		zap.String("env.GAE_INSTANCE", os.Getenv("GAE_INSTANCE")),
	)
	defer func() { zap.L().Info("Done.") }()

	// for example list envs
	// 2019-10-30 07:48:28 rakuten-items[20191030t134443]  2019-10-30T07:48:28.865Z	INFO	rakuten-items/main.go:73	List envs	{"envs": ["GAE_MEMORY_MB=256", "CGO_ENABLED=1", "GAE_INSTANCE=00c61b117ca010f5f64d25d702427d92cb718001dd263473a6bef4d4b49a186b5e", "PORT=8081", "HOME=/root", "GOROOT=/usr/local/go/", "GAE_SERVICE=rakuten-items", "REDISHOST=10.77.0.75", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin", "REDISPORT=6379", "GAE_DEPLOYMENT_ID=422093719274639499", "DEBIAN_FRONTEND=noninteractive", "GOOGLE_CLOUD_PROJECT=jpbay-253518", "GAE_ENV=standard", "PWD=/srv", "GAE_APPLICATION=u~jpbay-253518", "GAE_RUNTIME=go113", "GAE_VERSION=20191030t134443"]}

	if os.Getenv("GAE_VERSION") != "" {
		VERSION = os.Getenv("GAE_SERVICE") + "-" + os.Getenv("GAE_VERSION")
	}

	// Configure Stackdriver Tracing for gRPC
	// stackdriver
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               os.Getenv("GCLOUD_PROJECT"),
		MetricPrefix:            "acca-api",
		MonitoringClientOptions: []option.ClientOption{},
		TraceClientOptions:      []option.ClientOption{},
	})
	if err != nil {
		zap.L().Panic("Failed configure Stackdriver", zap.Error(err))
	}
	defer exporter.Flush()
	trace.RegisterExporter(exporter)
	if err := exporter.StartMetricsExporter(); err != nil {
		zap.L().Panic("Failed start metrics exporter", zap.Error(err))
	}
	defer exporter.StopMetricsExporter()

	zap.L().Info("Stackdriver Tracing - configured!")
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		zap.L().Panic("Failed register ocgrpc views", zap.Error(err))
	}

	sqlDB := setupPostgres(os.Getenv("PG_CONN"), 0, 5, 5)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(zap.L().Sugar().Debugf))
	_, err = db.Exec("SELECT version();")
	if err != nil {
		zap.L().Panic("Failed to check version to PostgreSQL.", zap.Error(err))
	}

	bqCl, err := bigquery.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		zap.L().Panic("Failed new client bigquery.", zap.Error(err))
	}

	firebaseApp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		zap.L().Panic("Failed get firebase client", zap.Error(err))
	}
	zap.L().Info("Firebase - configured!")

	fs, err := firebaseApp.Firestore(ctx)
	if err != nil {
		zap.L().Panic("Failed firebase app to firestore.", zap.Error(err))
	}
	defer fs.Close()

	sUpdater := updater.NewServer(nil)

	// аудитор http запросов (сохраняет в БД все реквесты и респонсы)
	httpAuditor := auditor.NewHttpAuditor(bqCl)
	defer httpAuditor.Stop()
	prometheus.MustRegister(httpAuditor)

	serv := services.NewService(db)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
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
	invServ := invoices.NewServer(db, fs)

	api.RegisterAccountsServer(grpcServer, accServ)
	api.RegisterInvoicesServer(grpcServer, invServ)
	api.RegisterUpdatesServer(grpcServer, sUpdater)

	// gRPC get port to listen
	portGrpc := os.Getenv("GRPC_PORT")
	if portGrpc == "" {
		portGrpc = "10002"
	}
	zap.L().Debug("gRPC - get port to listen", zap.String("got_port", portGrpc))

	// gRPC listen
	lis, err := net.Listen("tcp", ":"+portGrpc)
	if err != nil {
		zap.L().Panic("Failed to listen.", zap.Error(err))
	}
	zap.L().Info("gRPC - listen to address.", zap.String("address", lis.Addr().String()))

	// gRPC-Web server
	portGrpcWeb := os.Getenv("PORT")
	if portGrpcWeb == "" {
		portGrpcWeb = os.Getenv("GRPCWEB_PORT")
		if portGrpcWeb == "" {
			portGrpcWeb = "8081"
		}
	}
	zap.L().Debug("gRPC-WEB - get port to listen", zap.String("got_port", portGrpcWeb))

	httpServer := &http.Server{Addr: ":" + portGrpcWeb, Handler: httputils.GrpcWebHandlerFunc(ctx, grpcServer)}
	zap.L().Info("gRPC-WEB - will listen to address", zap.String("address", httpServer.Addr))

	// graceful stop
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		go func() {
			<-ctx.Done()
			grpcServer.Stop()

		}()
		grpcServer.GracefulStop()
		httpServer.Close()
	}()

	if *grpcReflectionF {
		// for debug via grpcurl
		reflection.Register(grpcServer)
	}

	zap.L().Info("gRPC-WEB server start")
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Error("GrpcWeb Serve error.", zap.Error(err))
		}
	}()

	// Запуск grpc сервера
	zap.L().Info("gRPC server start")
	if err := grpcServer.Serve(lis); err != nil {
		zap.L().Panic("Failed to serve.", zap.Error(err))
	}

	wg.Wait()
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
