package auditor

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
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

func (m *httpAuditMessage) Save() (map[string]bigquery.Value, string, error) {
	return map[string]bigquery.Value{
		"created_at": m.createdAt,
		"request_id": m.requestID,
		"method":     m.method,
		"user_id":    m.userID,
		"device_id":  m.deviceID,
		"body":       string(m.body),
		"user_ip":    m.ip,
		"proxy_ip":   m.proxyIP,
		"user_agent": m.userAgent,
		"session_id": m.sessionID,
	}, "", nil
}

type HttpAuditor struct {
	cl        *bigquery.Client
	toConvert chan *httpAuditMessage
	toInsert  chan *httpAuditMessage
	l         *zap.Logger
	wg        sync.WaitGroup

	mConvertLen     prometheus.Gauge
	mConvertCap     prometheus.Gauge
	mInsertLen      prometheus.Gauge
	mInsertCap      prometheus.Gauge
	mInsertSize     prometheus.Histogram
	mInsertDuration prometheus.Histogram
}

func NewHttpAuditor(cl *bigquery.Client) *HttpAuditor {
	a := &HttpAuditor{
		cl:        cl,
		toConvert: make(chan *httpAuditMessage, toConvertCap),
		toInsert:  make(chan *httpAuditMessage, toInsertCap),
		l:         zap.L().Named("auditor"),
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
		if len(messages) > 0 {
			a.insertBatch(messages)
		}
	}
}

func (a *HttpAuditor) insertBatch(messages []*httpAuditMessage) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer func() {
		cancel()
		d := time.Since(start)
		a.mInsertSize.Observe(float64(len(messages)))
		a.mInsertDuration.Observe(d.Seconds())
		a.l.Debug("Audit log messages inserted.", zap.Int("count", len(messages)), zap.Duration("duration", d))
	}()
	if err := a.cl.Dataset("activity").Table("http_requests").Inserter().Put(
		ctx,
		messages,
	); err != nil {
		a.l.Error("Failed to put audit log message.", zap.Error(err))
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
