package api

import (
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func MakeError(code codes.Code, msg string, details ...proto.Message) error {
	s := status.New(code, msg)
	if len(details) == 0 {
		// status.statusError (interface{ GRPCStatus() *Status })
		return s.Err()
	}

	// TODO ignore details in production unless debug tracing is enabled for current user

	sd, err := s.WithDetails(details...)
	if err != nil {
		return s.Err()
	}
	return sd.Err()
}
