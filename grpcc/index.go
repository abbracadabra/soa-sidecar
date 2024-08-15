package grpcc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"test/cluster"
	"test/connPool2/shared"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func ServeConnListener(ln net.Listener) {
	defer ln.Close()

	// listen port边界/ip边界; port边界较好，因为可能机器只有本地loopback
	localAddr := ln.Addr().(*net.TCPAddr)
	ip := localAddr.IP.String()
	port := localAddr.Port

	mh := myHandler{
		outbound: isOutbound(ip, port),
	}

	grpcServer := grpc.NewServer(
		grpc.UnknownServiceHandler(mh.handler),
	)

	defer grpcServer.GracefulStop()

	h2Server := &http2.Server{}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go h2Server.ServeConn(conn, &http2.ServeConnOpts{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				grpcServer.ServeHTTP(w, r)
			}),
		})

	}
}

// func main() {
// 	ln, err := net.Listen("tcp", ":8012")
// 	if err != nil {
// 		fmt.Println("Error setting up TCP listener:", err)
// 		return
// 	}
// 	defer ln.Close()

// 	fmt.Println("Listening on :8012")
// 	// server.Serve(ln)

// server := grpc.NewServer(
//	// ,grpc.Creds(creds)   tls server  https://github.com/devsu/grpc-proxy/blob/master/grpc-proxy.go
// 	grpc.UnknownServiceHandler(func(srv interface{}, serverStream grpc.ServerStream) error {
// 		return handler(srv, serverStream, true)
// 	}),
// )

// 	for {
// 		conn, err := ln.Accept()

// 		if err != nil {
// 			fmt.Println("Error accepting connection:", err)
// 			continue
// 		}

// 		// listen port边界/ip边界;
// 		localAddr := conn.LocalAddr()
// 		tcpAddr, ok := localAddr.(*net.TCPAddr)
// 		if !ok {
// 			conn.Close()
// 		}
// 		ip := tcpAddr.IP.String()
// 		port := tcpAddr.Port
// 		outbound := isOutbound(ip, port)

// 		if outbound {
// 			lis := &SingleConnListener{conn: conn}
// 			go server.Serve(lis)
// 		} else {
// 		}
// 	}
// }

func isOutbound(ip string, port int) bool {
	//判断ip边界 或 port边界
	return true
}

type myHandler struct {
	outbound bool
}

// 如果 streamHandler 返回一个 error，gRPC 服务器会将这个错误作为响应的一部分发送回客户端。具体来说，gRPC 会将错误转换为 gRPC 状态码和错误消息，然后返回给客户端
func (mh *myHandler) handler(srv interface{}, downstream grpc.ServerStream) error {

	downCtx := downstream.Context()
	fullMethodName, ok := grpc.MethodFromServerStream(downstream)
	if !ok {
		return status.Errorf(codes.Internal, "lowLevelServerStream not exists in context")
	}
	// We require that the director's returned context inherits from the serverStream.Context().

	md, _ := metadata.FromIncomingContext(downCtx) // header,是个copy
	// outCtx := metadata.NewOutgoingContext(serverStream.Context(), md)

	connLease, err := director(downCtx, md, mh.outbound)
	if err != nil {
		return err
	}
	defer connLease.Return()

	upCtx := metadata.NewOutgoingContext(downCtx, md) // 转发header
	upCtx, upCancel := context.WithCancel(upCtx)
	defer upCancel()
	// TODO(mwitkow): Add a `forwarded` header to metadata, https://en.wikipedia.org/wiki/X-Forwarded-For.
	// ctx可以操控让upstream关闭，这里ctx的parent是serverStream的ctx，serverStream完了upstream也完，ctx不影响conn
	upstream, err := grpc.NewClientStream(upCtx, &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}, connLease.GetConn(), fullMethodName) // 本身不产生网络传输
	if err != nil {
		return err
	}
	// Explicitly *do not close* s2cErrChan and c2sErrChan, otherwise the select below will not terminate.
	// Channels do not have to be closed, it is just a control flow mechanism, see
	// https://groups.google.com/forum/#!msg/golang-nuts/pZwdYRGxCIk/qpbHxRRPJdUJ
	s2cErrChan := forwardServerToClient(downstream, upstream)
	c2sErrChan := forwardClientToServer(upstream, downstream)
	// We don't know which side is going to stop sending first, so we need a select between the two.
	for i := 0; i < 2; i++ {
		select {
		case s2cErr := <-s2cErrChan:
			if s2cErr == io.EOF {
				// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore./
				// the clientStream>serverStream may continue pumping though.
				upstream.CloseSend()
			} else {
				// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need
				// to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and
				// exit with an error to the stack
				connLease.Unhealthy() // todo err区分upstream disconn or downstream disconn,upstream disconn才Unhealthy
				upCancel()
				return status.Errorf(codes.Internal, "failed proxying s2c: %v", s2cErr)
			}
		case c2sErr := <-c2sErrChan:
			// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two
			// cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers
			// will be nil.
			//trailer最后才能获取到
			//发送trailer
			downstream.SetTrailer(upstream.Trailer())
			if c2sErr == io.EOF {
				return nil
			}
			// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
			return c2sErr
		}
	}
	return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
}

func director(downCtx context.Context, md metadata.MD, outbound bool) (*shared.Lease, error) {
	cls := cluster.FindByName(md[":authority"][0], outbound) //集群
	ins := cls.Choose()                                      //实例  todo by 勇道
	if ins == nil {
		return nil, errors.New("no instance")
	}
	lease, err := ins.Pool.(*shared.Pool).Get(time.Second * 2) //连接
	if err != nil {
		return nil, err
	}
	return lease, nil

	// md, _ := metadata.FromIncomingContext(ctx)
	//复制header，向上游stream发送header
	// outCtx := metadata.NewOutgoingContext(ctx, md)

	// connectCtx, _ := context.WithTimeout(downCtx, time.Second*2)
	// cc, err := grpc.DialContext(connectCtx, "127.0.0.1:50051", grpc.WithCodec(codec.Codec()), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
	// 	Time:                10 * time.Second,
	// 	Timeout:             10 * time.Second,
	// 	PermitWithoutStream: true, // 允许没有活跃流的心跳
	// }))
	// return cc, err
}

func forwardClientToServer(src grpc.ClientStream, dst grpc.ServerStream) chan error {
	ret := make(chan error, 1)
	go func() {
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
		//protoimpl.UnknownFields 是 protobuf 实现的一部分
		//如果一个应用程序接收到一个包含未识别字段的消息，然后将该消息转发给其他服务，它可以保留这些未知字段并将它们包括在转发的消息中
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
