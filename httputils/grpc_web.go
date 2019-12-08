package httputils

import (
	"context"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
)

// GrpcWebHandlerFunc возвращает хандлер для grpc-web
func GrpcWebHandlerFunc(ctx context.Context, s *grpc.Server) http.HandlerFunc {
	zap.L().Info("GrpcWeb server starting.")

	opts := []grpcweb.Option{
		// gRPC-Web compatibility layer with CORS configured to accept on every request
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
	}
	wrappedGrpc := grpcweb.WrapServer(s, opts...)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if wrappedGrpc.IsAcceptableGrpcCorsRequest(req) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "x-grpc-web, access-token, Accept, Content-Type, Content-Length, Accept-Encoding")
			return
		}
		if wrappedGrpc.IsGrpcWebSocketRequest(req) || wrappedGrpc.IsGrpcWebRequest(req) {
			wrappedGrpc.ServeHTTP(w, req)
			return
		}

		http.DefaultServeMux.ServeHTTP(w, req)
	})
}
