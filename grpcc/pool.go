package grpcc

import (
	"context"
	"strconv"
	"test/cluster"
	"test/codec"
	"test/connPool/shared"
	"test/localInstance"
	"test/types"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func PoolFactoryOut(cls *cluster.Cluster, ins *cluster.Instance) (types.Closable, error) {
	// todo get service info/config
	p := shared.NewPool(1, 1, 9999, time.Minute*1, func() (interface{}, error) {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		cc, err := grpc.DialContext(ctx, ins.IP+":"+strconv.Itoa(ins.Port), grpc.WithCodec(codec.Codec()), grpc.WithInsecure())
		if err != nil {
			return nil, err
		}

		return cc, nil

	}, func(conn interface{}) {
		conn.(*grpc.ClientConn).Close()
	})
	return p, nil
}

func PoolFactoryIn(ins *localInstance.LocalInstance) (interface{}, error) {
	p := shared.NewPool(1, 1, 9999, time.Minute*1, func() (interface{}, error) {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
		cc, err := grpc.DialContext(ctx, ins.Ip+":"+strconv.Itoa(ins.Port), grpc.WithCodec(codec.Codec()), grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true, // 允许没有活跃流的心跳
		}))
		if err != nil {
			return nil, err
		}

		return cc, nil

	}, func(conn interface{}) {
		conn.(*grpc.ClientConn).Close()
	})
	return p, nil
}

//grpc.WithKeepaliveParams(keepalive.ClientParameters{
//Time:                10 * time.Second,
//Timeout:             3 * time.Second,
//PermitWithoutStream: true, // 允许没有活跃流的心跳
//})
