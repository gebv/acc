package auditor

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	toConvertCap  = 1024
	toInsertCap   = 1024
	maxBatch      = 8192
	maxBatchDelay = time.Second
)

type httpAuditMessage struct {
	userID    *int64
	ip        *string
	proxyIP   *string
	userAgent string
	deviceID  string
	sessionID *string
	requestID string
	method    string
	payload   interface{}
	createdAt time.Time

	body []byte
}

type HttpAuditor struct {
	db          *sql.DB
	toConvert   chan *httpAuditMessage
	toInsert    chan *httpAuditMessage
	l           *zap.Logger
	insertQuery string
	wg          sync.WaitGroup

	mConvertLen     prometheus.Gauge
	mConvertCap     prometheus.Gauge
	mInsertLen      prometheus.Gauge
	mInsertCap      prometheus.Gauge
	mInsertSize     prometheus.Histogram
	mInsertDuration prometheus.Histogram
}

func NewHttpAuditor(db *sql.DB) *HttpAuditor {
	q := `INSERT INTO activity.http_requests (
		created_at,
		request_id,
		method,
		user_id,
		device_id,
		body,
		user_ip,
		proxy_ip,
		user_agent,
		session_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	a := &HttpAuditor{
		db:          db,
		toConvert:   make(chan *httpAuditMessage, toConvertCap),
		toInsert:    make(chan *httpAuditMessage, toInsertCap),
		l:           zap.L().Named("auditor"),
		insertQuery: q,
		mConvertLen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "auditor_convert_len",
			Help: "Length of internal convert channel.",
		}),
		mConvertCap: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "auditor_convert_cap",
			Help: "Capacity of internal convert channel.",
		}),
		mInsertLen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "auditor_insert_len",
			Help: "Length of internal insert channel.",
		}),
		mInsertCap: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "auditor_insert_cap",
			Help: "Capacity of internal insert channel.",
		}),
		mInsertSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "auditor_insert_size_rows",
			Help:    "Size of a single batch insert.",
			Buckets: prometheus.ExponentialBuckets(maxBatch/32, 2, 5),
		}),
		mInsertDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "auditor_insert_duration_seconds",
			Help:    "Duration of a single batch insert.",
			Buckets: prometheus.ExponentialBuckets(maxBatchDelay.Seconds()/32, 2, 5),
		}),
	}

	a.l.Info("Started.")
	a.wg.Add(2)
	go a.runConverter()
	go a.runInserter()
	return a
}

func (a *HttpAuditor) Stop() {
	close(a.toConvert)
	a.wg.Wait()
	a.l.Info("Stopped.")
}

func (a *HttpAuditor) runConverter() {
	defer a.wg.Done()

	var err error
	for m := range a.toConvert {
		m.body, err = json.Marshal(m.payload)
		if err != nil {
			a.l.Error("Failed to marshal audit log message to JSON.", zap.Error(err))
			continue
		}
		m.payload = nil

		a.toInsert <- m
	}

	close(a.toInsert)
}

func (a *HttpAuditor) runInserter() {
	defer a.wg.Done()
	t := time.NewTicker(maxBatchDelay)
	defer t.Stop()

	var exit bool
	for !exit {
		// collect batch up to maxBatch messages and up to maxBatchDelay seconds
		messages := make([]*httpAuditMessage, 0, maxBatch)
		var insert bool
		for !insert {
			select {
			case m := <-a.toInsert:
				if m == nil {
					exit = true
					insert = true
					break
				}

				messages = append(messages, m)
				if len(messages) == maxBatch {
					insert = true
				}

			case <-t.C:
				insert = true
			}
		}

		a.insertBatch(messages)
	}
}

func (a *HttpAuditor) insertBatch(messages []*httpAuditMessage) {
	start := time.Now()
	tx, err := a.db.Begin()
	if err != nil {
		a.l.Error("Failed to begin transaction.", zap.Error(err))
		return
	}

	defer func() {
		if err = tx.Commit(); err != nil {
			a.l.Error("Failed to commit transaction.", zap.Error(err))
			return
		}
		d := time.Since(start)
		a.mInsertSize.Observe(float64(len(messages)))
		a.mInsertDuration.Observe(d.Seconds())
		a.l.Debug("Audit log messages inserted.", zap.Int("count", len(messages)), zap.Duration("duration", d))
	}()

	stmt, err := tx.Prepare(a.insertQuery)
	if err != nil {
		a.l.Error("Failed to prepare statement.", zap.Error(err))
		return
	}
	defer func() {
		if err = stmt.Close(); err != nil {
			a.l.Error("Failed to close statement.", zap.Error(err))
		}
	}()

	for _, m := range messages {
		if _, err = stmt.Exec(
			m.createdAt,
			m.requestID,
			m.method,
			m.userID,
			m.deviceID,
			string(m.body),
			m.ip,
			m.proxyIP,
			m.userAgent,
			m.sessionID,
		); err != nil {
			a.l.Error("Failed to insert audit log message.", zap.Error(err))
		}
	}
}

func (a *HttpAuditor) Log(
	ctx context.Context,
	userID *int64,
	deviceID string,
	sessionID string,
	userAgent string,
	ip string,
	proxyIP string,
	requestID string,
	method string,
	payload interface{},
) {
	msg := &httpAuditMessage{
		userID:    userID,
		deviceID:  deviceID,
		userAgent: userAgent,
		createdAt: time.Now(),
		requestID: requestID,
		method:    method,
		payload:   payload,
	}

	if sessionID != "" {
		msg.sessionID = &sessionID
	}

	if ip != "" {
		msg.ip = &ip
	}

	if proxyIP != "" {
		msg.proxyIP = &proxyIP
	}

	a.toConvert <- msg
}

func (a *HttpAuditor) Describe(ch chan<- *prometheus.Desc) {
	a.mConvertLen.Describe(ch)
	a.mConvertCap.Describe(ch)
	a.mInsertLen.Describe(ch)
	a.mInsertCap.Describe(ch)
	a.mInsertSize.Describe(ch)
	a.mInsertDuration.Describe(ch)
}

func (a *HttpAuditor) Collect(ch chan<- prometheus.Metric) {
	a.mConvertLen.Set(float64(len(a.toConvert)))
	a.mConvertCap.Set(float64(cap(a.toConvert)))
	a.mInsertLen.Set(float64(len(a.toInsert)))
	a.mInsertCap.Set(float64(cap(a.toInsert)))

	a.mConvertLen.Collect(ch)
	a.mConvertCap.Collect(ch)
	a.mInsertLen.Collect(ch)
	a.mInsertCap.Collect(ch)
	a.mInsertSize.Collect(ch)
	a.mInsertDuration.Collect(ch)
}

// check interfaces
var (
	_ prometheus.Collector = (*HttpAuditor)(nil)
)
