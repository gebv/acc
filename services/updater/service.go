package updater

import (
	"fmt"
	"strings"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/services"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	nc *nats.EncodedConn
	l  *zap.Logger
}

func NewServer(nc *nats.EncodedConn) *Server {
	s := &Server{
		nc: nc,
		l:  zap.L().Named("updater"),
	}

	return s
}

func (s *Server) GetUpdate(req *api.GetUpdateRequest, stream api.Updates_GetUpdateServer) error {
	clientID := services.GetClient(stream.Context()).ClientID

	var chErr = make(chan error)
	s.l.Info("Subscribed.", zap.Int64("client_id", clientID))
	sub, err := s.nc.Subscribe(fmt.Sprintf("client.%d.>", clientID), func(m *Update) {
		if err := stream.Send(convertUpdate(m)); err != nil {
			chErr <- err
		}
	})
	if err != nil {
		return errors.Wrap(err, "Failed subscribe.")
	}
	defer func() {
		s.l.Info("Unsubscribed.", zap.Int64("client_id", clientID))
		sub.Unsubscribe()
	}()

	for {
		err := <-chErr
		// grpc status
		gs, ok := status.FromError(err)
		if ok {
			if gs.Code() == codes.Internal && gs.Message() == "transport is closing" {
				return api.MakeError(codes.Aborted, "Transport is closing.")
			}
			s.l.Warn("stream.Send failed: info on error from grpc status",
				zap.Error(err),
				zap.Any("grpc_code", gs.Code()),
				zap.Any("grpc_message", gs.Message()),

				zap.Int64("client_id", clientID),
			)
		}

		if strings.Contains(err.Error(), "transport is closing") {
			// https://github.com/grpc/grpc-go/blob/9e7c1463564add763b262504eabda61fde9c3f1d/internal/transport/transport.go#L698
			return api.MakeError(codes.Aborted, "Transport is closing.")
		}

		if strings.Contains(err.Error(), "the stream is done or WriteHeader was already called") {
			// https:github.com/grpc/grpc-go/blob/3fc743058b25bc974180a7a61656554e31f92635/internal/transport/http2_server.go#L54
			// Смотри где эта ошибка встречается
			// https://github.com/grpc/grpc-go/search?utf8=%E2%9C%93&q=ErrIllegalHeaderWrite&type=
			// - либо если streamDone
			// - либо isHeaderSent
			return api.MakeError(codes.Aborted, "Stream is done or header sent (not exact wording).")
		}

		s.l.Warn("stream.Send: ", zap.Error(err), zap.Int64("client_id", clientID))
		return errors.Wrap(err, "Failed to send update.")
	}

	return nil
}

func SubjectFromInvoice(clientID *int64, invoiceID int64) string {
	if clientID == nil {
		return fmt.Sprintf("client.nil.invoice.%d", invoiceID)
	}
	return fmt.Sprintf("client.%d.invoice.%d", *clientID, invoiceID)
}

func SubjectFromTransaction(clientID *int64, transactionID int64) string {
	if clientID == nil {
		return fmt.Sprintf("client.nil.transaction.%d", transactionID)
	}
	return fmt.Sprintf("client.%d.transaction.%d", *clientID, transactionID)
}
