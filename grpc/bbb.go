package main

import (
	"context"
	"fmt"
	"io"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Serve(conn net.Conn) error {
	var server = grpc.NewServer(
		grpc.UnknownServiceHandler(handler),
	)
	// ,grpc.Creds(creds)   tls server  https://github.com/devsu/grpc-proxy/blob/master/grpc-proxy.go
	server.Serve(&newSingleConnListener{conn: conn})
	return nil
}

func director(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	outCtx := metadata.NewOutgoingContext(ctx, md.Copy())
	//md[":authority"]

	// cc, err := grpc.Dial(target, grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
	// 	return (&net.Dialer{}).DialContext(ctx, "tcp", target)
	// }), grpc.WithInsecure())
	// ctx只影响dialing过程
	cc, err := grpc.DialContext(ctx, "api-service.prod.svc.local", grpc.WithCodec(Codec()))

	// cc, err := grpc.NewClient(gclient.NewClientConfig{
	// 	Target:      target,
	// 	Options:     []grpc.DialOption{grpc.WithInsecure()},
	// 	DialContext: ctx,
	// })
	return outCtx, cc, err
}

// 如果 streamHandler 返回一个 error，gRPC 服务器会将这个错误作为响应的一部分发送回客户端。具体来说，gRPC 会将错误转换为 gRPC 状态码和错误消息，然后返回给客户端
func handler(srv interface{}, serverStream grpc.ServerStream) error {
	// little bit of gRPC internals never hurt anyone
	fullMethodName, ok := grpc.MethodFromServerStream(serverStream)
	if !ok {
		return status.Errorf(codes.Internal, "lowLevelServerStream not exists in context")
	}
	// We require that the director's returned context inherits from the serverStream.Context().
	outgoingCtx, backendConn, err := director(serverStream.Context(), fullMethodName)
	backendConn.Invoke()
	if err != nil {
		return err
	}

	//defer return back backendConn

	clientCtx, clientCancel := context.WithCancel(outgoingCtx)
	defer clientCancel()
	// TODO(mwitkow): Add a `forwarded` header to metadata, https://en.wikipedia.org/wiki/X-Forwarded-For.
	// ctx可以操控让clientStream关闭，这里ctx的parent是serverStream的ctx，serverStream完了clientStream也完，ctx不影响conn
	clientStream, err := grpc.NewClientStream(clientCtx, &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}, backendConn, fullMethodName)
	if err != nil {
		// backendConn.Close()
		return err
	}
	// Explicitly *do not close* s2cErrChan and c2sErrChan, otherwise the select below will not terminate.
	// Channels do not have to be closed, it is just a control flow mechanism, see
	// https://groups.google.com/forum/#!msg/golang-nuts/pZwdYRGxCIk/qpbHxRRPJdUJ
	s2cErrChan := forwardServerToClient(serverStream, clientStream)
	c2sErrChan := forwardClientToServer(clientStream, serverStream)
	// We don't know which side is going to stop sending first, so we need a select between the two.
	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>serverStream may continue pumping though.
				clientStream.CloseSend()
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				clientCancel()
				return status.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two
			// cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers
			// will be nil.
			serverStream.SetTrailer(clientStream.Trailer())
			if c2sErr == io.EOF {
				return nil
			}
			// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
			return c2sErr
		}
	}
	return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

func forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
		// 卧槽，这里要的是业务的pb类型！！！！！！！！！
		f := &emptypb.Empty{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if i == 0 {
				// This is a bit of a hack, but client to server headers are only readable after first client msg is
				// received but must be written to server stream before the first msg is flushed.
				// This is the only place to do it nicely.
				md, err := src.Header()
				if err != nil {
					ret <- err
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- err
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

func forwardServerToClient(src grpc.ServerStream, dst grpc.ClientStream) chan error {
	ret := make(chan error, 1)
	go func() {
		f := &emptypb.Empty{}
		for i := 0; ; i++ {
			if err := src.RecvMsg(f); err != nil {
				ret <- err // this can be io.EOF which is happy case
				break
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- err
				break
			}
		}
	}()
	return ret
}

type newSingleConnListener struct {
	conn net.Conn
	done bool
}

func (l *newSingleConnListener) Accept() (net.Conn, error) {
	if l.done {
		return nil, io.EOF
	}
	l.done = true
	return l.conn, nil
}

func (l *newSingleConnListener) Close() error {
	return l.conn.Close()
}

func (l *newSingleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

type protoCodec struct{}

func (protoCodec) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (protoCodec) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func (protoCodec) String() string {
	return "proto"
}

func CodecWithParent(fallback grpc.Codec) grpc.Codec {
	return &rawCodec{fallback}
}

type rawCodec struct {
	parentCodec grpc.Codec
}

type frame struct {
	payload []byte
}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*frame)
	if !ok {
		return c.parentCodec.Marshal(v)
	}
	return out.payload, nil

}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*frame)
	if !ok {
		return c.parentCodec.Unmarshal(data, v)
	}
	dst.payload = data
	return nil
}

func (c *rawCodec) String() string {
	return fmt.Sprintf("proxy>%s", c.parentCodec.String())
}

func Codec() grpc.Codec {
	return CodecWithParent(&protoCodec{})
}
