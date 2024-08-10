package codec

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func Codec() grpc.Codec {
	return CodecWithParent(&protoCodec{})
}

func CodecWithParent(fallback grpc.Codec) grpc.Codec {
	return &rawCodec{fallback}
}

type rawCodec struct {
	parentCodec grpc.Codec
}

type myMessage struct {
	payload []byte
}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.(*myMessage)
	if !ok {
		fmt.Println("use default codec")
		return c.parentCodec.Marshal(v)
	}
	fmt.Println("use raw payload")
	return out.payload, nil

}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*myMessage)
	if !ok {
		return c.parentCodec.Unmarshal(data, v)
	}
	dst.payload = data
	return nil
}

func (c *rawCodec) String() string {
	return fmt.Sprintf("proxy>%s", c.parentCodec.String())
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
