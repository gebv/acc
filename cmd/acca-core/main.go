package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	firebase "firebase.google.com/go"
	_ "github.com/lib/pq"
	_ "github.com/solcates/go-sql-bigquery"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/unix"
	"google.golang.org/api/option"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/gebv/acca/engine"
	_ "github.com/gebv/acca/engine/strategies/invoices/refund"
	_ "github.com/gebv/acca/engine/strategies/invoices/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/moedelo"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank"
	_ "github.com/gebv/acca/engine/strategies/transactions/sberbank_refund"
	_ "github.com/gebv/acca/engine/strategies/transactions/simple"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe"
	_ "github.com/gebv/acca/engine/strategies/transactions/stripe_refund"
	"github.com/gebv/acca/engine/worker"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/provider/stripe"
	"github.com/gebv/acca/services"
)

var (
	VERSION         = "dev"
	pgConnF         = flag.String("pg-conn", "postgres://acca:acca@127.0.0.1:5433/acca?sslmode=disable", "PostgreSQL connection string.")
	grpcAddrsF      = flag.String("grpc-addrs", "127.0.0.1:10011", "gRPC listen address.")
	webhookAddrsF   = flag.String("webhook-addrs", "127.0.0.1:10003", "HTTP webhooks listen address.")
	grpcReflectionF = flag.Bool("grpc-reflection", false, "Enable gRPC reflection.")
	genAccessTokenF = flag.Bool("gen-access-token", false, "access_token generation.")
	currencyF       = flag.String("currency", "rub", "currency from acccess_token generation.")
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

	sqlDB := setupPostgres(*pgConnF, 0, 5, 5)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(zap.L().Sugar().Debugf))
	_, err = db.Exec("SELECT version();")
	if err != nil {
		zap.L().Panic("Failed to check version to PostgreSQL.", zap.Error(err))
	}

	if *genAccessTokenF {
		zap.L().Info("Generation Access Token.")
		client := services.NewClient()
		if err := db.Save(client); err != nil {
			zap.L().Error("Failed save client.", zap.Error(err))
		}
		am := engine.NewAccountManager(db)
		if err := am.UpsertCurrency(client.ClientID, *currencyF, nil); err != nil {
			zap.L().Error("Failed save currency.", zap.Error(err))
		}
		curr, err := am.GetCurrency(client.ClientID, *currencyF)
		if err != nil {
			zap.L().Error("Failed load currency.", zap.Error(err))
		}
		accID, err := am.CreateAccount(client.ClientID, curr.CurrencyID, "system", nil)
		if err != nil {
			zap.L().Error("Failed create system account.", zap.Error(err))
		}
		conf := map[string]interface{}{
			"currentcy":         *currencyF,
			"system_account_id": accID,
			"access_token":      client.AccessToken,
		}
		if err := json.NewEncoder(os.Stdout).Encode(conf); err != nil {
			panic(err)
		}
		return
	}

	//pb, err := pubsub.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	//if err != nil {
	//	zap.L().Panic("Failed get pubsub client", zap.Error(err))
	//}
	//zap.L().Info("PubSub - configured!")

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

	var moeDeloProvider *moedelo.Provider
	if os.Getenv("MOEDELO_ENTRYPOINT_URL") != "" {
		moeDeloProvider = moedelo.NewProvider(
			db,
			moedelo.Config{
				EntrypointURL: os.Getenv("MOEDELO_ENTRYPOINT_URL"),
				Token:         os.Getenv("MOEDELO_TOKEN"),
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

	//worker.Run(fs, db, sberProvider, moeDeloProvider, stripeProvider) TODO использовать после правок в провыйдерах сбера и моедело
	_ = sberProvider
	_ = moeDeloProvider
	worker.Run(ctx, fs, db, nil, nil, stripeProvider)

	var wg sync.WaitGroup

	if moeDeloProvider != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// TODO раскомментировать после провок
			//moeDeloProvider.RunCheckStatusListener(ctx)
		}()
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		go func() {
			defer wg.Done()
			<-ctx.Done()
		}()
	}()

	wg.Wait()
	zap.L().Info("Done")
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
