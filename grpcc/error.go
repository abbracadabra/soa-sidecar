package grpcc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	//"google.golang.org/grpc/status"
	//"google.golang.org/grpc/internal/status"
)

//type GatewayError struct {
//	error
//	code codes.Code
//}
//
//func (e *GatewayError) Code() codes.Code {
//	return e.code
//}
//
//func NewGatewayError(code codes.Code, msg string) *GatewayError {
//	return &GatewayError{
//		code:  code,
//		error: errors.New(msg),
//	}
//}

func toGrpcStatusError(err error) error {
	if _, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return err
	}
	return status.Errorf(codes.Internal, err.Error(), err)
}

// for Error Model,return grpcStatusError https://cloud.google.com/apis/design/errors
//type grpcStatusError struct {
//	error
//	grpcStatus *status.Status
//}
//
//func (e *grpcStatusError) GRPCStatus() *status.Status { return e.grpcStatus }
//func (e *grpcStatusError) Error() string {
//	return e.grpcStatus.Code().String() + "_" + e.grpcStatus.Message()
//}
