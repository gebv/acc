package main

import (
	"context"
	"database/sql"
	"flag"
	"math/rand"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"contrib.go.opencensus.io/exporter/stackdriver"
	firebase "firebase.google.com/go"
	"github.com/labstack/echo"
	echo_middleware "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/api/option"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	_ "github.com/gebv/acca/engine/strategies/invoices/refund"
	_ "github.com/gebv/acca/engine/strategies/invoices/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/moedelo"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank_refund"
	_ "github.com/gebv/acca/engine/strategies/transactions/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe_refund"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/provider/stripe"
	"github.com/gebv/acca/services/auditor"
)

var (
	VERSION = "dev"

	onLoggerDev         = flag.Bool("logger-dev", false, "Enable development logger.")
	onLoggerDebugLevelF = flag.Bool("logger-debug-level", false, "Enable debug level logger.")
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

	// аудитор http запросов (сохраняет в БД все реквесты и респонсы)
	httpAuditor := auditor.NewHttpAuditor(bqCl)
	defer httpAuditor.Stop()
	prometheus.MustRegister(httpAuditor)

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

	var sberProvider *sberbank.Provider
	if os.Getenv("SBERBANK_ENTRYPOINT_URL") != "" {
		sberProvider = sberbank.NewProvider(
			db,
			sberbank.Config{
				EntrypointURL: os.Getenv("SBERBANK_ENTRYPOINT_URL"),
				Token:         os.Getenv("SBERBANK_TOKEN"),
				Password:      os.Getenv("SBERBANK_PASSWORD"),
				UserName:      os.Getenv("SBERBANK_USER_NAME"),
			},
			nil,
		)
	}

	var stripeProvider *stripe.Provider
	if os.Getenv("STRIPE_KEY") != "" {
		stripeProvider = stripe.NewProvider(
			db,
			fs,
		)
	}

	// Web server
	portWeb := os.Getenv("PORT")
	if portWeb == "" {
		portWeb = os.Getenv("WEB_PORT")
		if portWeb == "" {
			portWeb = "8081"
		}
	}
	zap.L().Debug("WEB - get port to listen", zap.String("got_port", portWeb))

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
	e.Use(echo_middleware.BodyLimit("64K"))

	if sberProvider != nil {
		e.GET("/webhook/sberbank", sberProvider.WebhookHandler())
	}
	if stripeProvider != nil {
		e.POST("/webhook/stripe", stripeProvider.WebhookHandler())
	}

	wg.Add(1)
	go func() {
		zap.L().Info("start server sberbank webhook ",
			zap.String("address", ":"+portWeb),
			zap.Strings("paths", []string{
				"/webhook/sberbank",
				"/webhook/stripe",
			}),
		)
		if err := e.Start(":" + portWeb); err != nil {
			zap.L().Error("failed run server webhooks", zap.Error(err))
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		zap.L().Info("Stoped webhook, sleep 3 sec\n")
		time.Sleep(3 * time.Second) // Слип из-за тестов, не успивает прийти последнее сообщение по webhook
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
