package httputils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type ctxKey int

const (
	requestInfoCtxKey ctxKey = iota
)

// SetRequestInfo returns a new context with set (or re-set) RequestInfo.
func SetRequestInfo(ctx context.Context, appVersion string) (out context.Context, res RequestInfo) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md["x-forwarded-for"]) > 0 {
			ipsl := strings.Split(md["x-forwarded-for"][0], ", ")
			res.RealIP = ipsl[0]
			if len(ipsl) > 1 {
				res.ProxyIPs = ipsl[1:]
			}
		}
		if len(md["user-agent"]) > 0 {
			res.UserAgent = md["user-agent"][0]
		}

		if len(md["device-id"]) > 0 {
			res.DeviceID = md["device-id"][0]
		}

		if len(md["session-id"]) > 0 {
			res.SessionID = md["session-id"][0]
		}

		if len(md["x-request-id"]) > 0 {
			res.RequestID = md["x-request-id"][0]
		}
	}
	if res.RealIP == "" {
		p, ok := peer.FromContext(ctx)
		if ok {
			res.ProxyIPs = []string{strings.Split(p.Addr.String(), ":")[0]}
		}
	}

	if res.RequestID == "" {
		res.RequestID = appCreatedRequestID()
	}
	res.AppVersion = appVersion

	out = context.WithValue(ctx, requestInfoCtxKey, res)

	return out, res
}

// GetRequestInfo returns RequestInfo from the context.
func GetRequestInfo(ctx context.Context) (res RequestInfo) {
	return ctx.Value(requestInfoCtxKey).(RequestInfo)
}

// RequestInfo контейнер с мета-информацией о реквесте.
type RequestInfo struct {
	RealIP     string
	ProxyIPs   []string
	DeviceID   string
	SessionID  string
	UserAgent  string
	RequestID  string
	AppVersion string
}

func (ri RequestInfo) FirstProxyIP() string {
	if len(ri.ProxyIPs) > 0 {
		return ri.ProxyIPs[0]
	}
	return ""
}

// application created
// ac-2006-01-02T15:04:05.000-XXX###XXX
func appCreatedRequestID() string {
	return "ac-" + time.Now().Format("2006-01-02T15:04:05.000") + randString(9)
}

func randString(len int) string {
	b := make([]byte, len)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
