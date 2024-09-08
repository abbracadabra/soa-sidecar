package httpp

import "errors"

type GatewayError struct {
	error
	code int
}

func (e *GatewayError) Code() int {
	return e.code
}

func NewGatewayError(code int, msg string) *GatewayError {
	return &GatewayError{
		code:  code,
		error: errors.New(msg),
	}
}
