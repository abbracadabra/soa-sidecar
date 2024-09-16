package grpcc

import (
	"context"
	"errors"
	"io"
	"net"
	"test/cluster"
	"test/connPool/shared"
	"test/localInstance"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"test/utils/servNameUtil"
)

var (
	desc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}
)

func init() {
	cluster.RegisterPoolFactoryOut("grpc", PoolFactoryOut)
	cluster.RegisterLoadBalancerFactory("grpc", cluster.NewDefaultLoadBalancer)
}

// https://github.com/mwitkow/grpc-proxy/blob/master/proxy/handler.go
func ServeListenerIn(ln net.Listener, servName string, ins *localInstance.LocalInstance) {
	defer ln.Close()
	var grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(createHandle(&phaseHookIn{ins: ins})),
	)
	grpcServer.Serve(ln)
}

func ServeListenerOut(ln net.Listener) {
	defer ln.Close()
	var grpcServer = grpc.NewServer(
		grpc.UnknownServiceHandler(createHandle(&phaseHookOut{})),
	)
	grpcServer.Serve(ln)
}

func createHandle(phaseHook phaseHook) grpc.StreamHandler {
	// 如果 streamHandler 返回一个 error，gRPC 服务器会将这个错误作为响应的一部分发送回客户端。具体来说，gRPC 会将错误转换为 gRPC 状态码和错误消息，然后返回给客户端
	return func(server interface{}, downstream grpc.ServerStream) error {

		downCtx := downstream.Context()
		fullMethodName, ok := grpc.MethodFromServerStream(downstream)
		if !ok {
			return status.Errorf(codes.Internal, "lowLevelServerStream not exists in context")
		}
		// We require that the director's returned context inherits from the serverStream.Context().

		md, _ := metadata.FromIncomingContext(downCtx) // get header copy

		err := phaseHook.filter(md)
		if err != nil {
			return toGrpcStatusError(err)
		}

		connLease, err := phaseHook.director(downCtx, md)
		if err != nil {
			return toGrpcStatusError(err)
		}
		defer connLease.Return()

		upCtx := metadata.NewOutgoingContext(downCtx, md) // 转发header
		upCtx, upCancel := context.WithCancel(upCtx)
		defer upCancel()
		// ctx可以操控让upstream关闭，这里ctx的parent是serverStream的ctx，serverStream完了upstream也完，ctx不影响conn
		upstream, err := grpc.NewClientStream(upCtx, desc, connLease.GetConn().(*grpc.ClientConn), fullMethodName) // 本身不产生网络传输
		if err != nil {
			return toGrpcStatusError(err)
		}
		// Explicitly *do not close* s2cErrChan and c2sErrChan, otherwise the select below will not terminate. Channels do not have to be closed, it is just a control flow mechanism, see https://groups.google.com/forum/#!msg/golang-nuts/pZwdYRGxCIk/qpbHxRRPJdUJ
		d2uErrChan := forwardDown2Up(downstream, upstream)
		u2dErrChan := forwardUp2Down(upstream, downstream)
		// We don't know which side is going to stop sending first, so we need a select between the two.
		for i := 0; i < 2; i++ {
			select {
			case d2uErr := <-d2uErrChan:
				err := d2uErr[0]
				isUpstreamErr := d2uErr[1].(bool)
				if err == io.EOF {
					// this is the happy case where the sender has encountered io.EOF, and won't be sending anymore.the clientStream>serverStream may continue pumping though.
					upstream.CloseSend()
				} else {
					// however, we may have gotten a receive error (stream disconnected, a read error etc) in which case we need to cancel the clientStream to the backend, let all of its goroutines be freed up by the CancelFunc and exit with an error to the stack
					if isUpstreamErr {
						connLease.Unhealthy() // todo grpc会自动重连，是否需要删掉所有Unhealthy、upCancel
					}
					upCancel()
					return status.Errorf(codes.Internal, "failed proxying d2u: %v", err)
				}
			case u2dErr := <-u2dErrChan:
				err := u2dErr[0].(error)
				isUpstreamErr := u2dErr[1].(bool)
				// This happens when the clientStream has nothing else to offer (io.EOF), returned a gRPC error. In those two cases we may have received Trailers as part of the call. In case of other errors (stream closed) the trailers will be nil.
				//trailer最后才能获取到
				//发送trailer
				downstream.SetTrailer(upstream.Trailer())
				if err == io.EOF {
					return nil
				}
				if isUpstreamErr {
					connLease.Unhealthy()
				}
				// c2sErr will contain RPC error from client code. If not io.EOF return the RPC error as server stream error.
				return err
			}
		}
		return status.Errorf(codes.Internal, "gRPC proxying should never reach this stage.")
	}
}

type phaseHook interface {
	director(downCtx context.Context, md metadata.MD) (*shared.Lease, error)
	filter(metadata.MD) error
}
type phaseHookOut struct {
	phaseHook
}

func (*phaseHookOut) director(downCtx context.Context, md metadata.MD) (*shared.Lease, error) {
	servName := servNameUtil.ExtractServName(md[":authority"][0])
	cls := cluster.GetOrCreate(servName) //集群
	//筛泳道、再lb  //key、lane、set
	ins := cls.Choose(&cluster.RouteInfo{Color: md["lane"][0]}) //实例
	if ins == nil {
		return nil, errors.New("no instance")
	}
	lease, err := ins.Pool.(*shared.Pool).Get(time.Second * 2) //连接
	if err != nil {
		return nil, err
	}
	return lease, nil
}
func (*phaseHookOut) filter(md metadata.MD) error {
	return nil
}

type phaseHookIn struct {
	phaseHook
	ins *localInstance.LocalInstance
}

func (c *phaseHookIn) director(downCtx context.Context, md metadata.MD) (*shared.Lease, error) {
	lease, err := c.ins.Pool.(*shared.Pool).Get(time.Second * 2)
	if err != nil {
		return nil, err
	}
	return lease, nil
}
func (c *phaseHookIn) filter(md metadata.MD) error {
	return nil
}

func forwardUp2Down(src grpc.ClientStream, dst grpc.ServerStream) chan [2]any {
	ret := make(chan [2]any, 1)
	go func() {
		var f []byte
		for i := 0; ; i++ {
			if err := src.RecvMsg(&f); err != nil {
				ret <- [2]any{err, true} // this can be io.EOF which is happy case
				break
			}
			if i == 0 {
				// This is a bit of a hack, but client to server headers are only readable after first client msg is received but must be written to server stream before the first msg is flushed. This is the only place to do it nicely.
				md, err := src.Header()
				if err != nil {
					ret <- [2]any{err, true}
					break
				}
				if err := dst.SendHeader(md); err != nil {
					ret <- [2]any{err, false}
					break
				}
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- [2]any{err, false}
				break
			}
		}
	}()
	return ret
}

func forwardDown2Up(src grpc.ServerStream, dst grpc.ClientStream) chan [2]any {
	ret := make(chan [2]any, 1)
	go func() {
		//protoimpl.UnknownFields 是 protobuf 实现的一部分
		//如果一个应用程序接收到一个包含未识别字段的消息，然后将该消息转发给其他服务，它可以保留这些未知字段并将它们包括在转发的消息中
		var f []byte
		for i := 0; ; i++ {
			if err := src.RecvMsg(&f); err != nil {
				ret <- [2]any{err, false} // this can be io.EOF which is happy case
				break
			}
			if err := dst.SendMsg(f); err != nil {
				ret <- [2]any{err, true}
				break
			}
		}
	}()
	return ret
}
