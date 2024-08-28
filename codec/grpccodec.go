package codec

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// proto编解码器
func Codec() grpc.Codec {
	return &rawCodec{}
}

type rawCodec struct {
}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
	out, ok := v.([]byte)
	if ok {
		return out, nil
	}
	pt, ok := v.(proto.Message)
	if ok {
		return proto.Marshal(pt)
	}
	return nil, fmt.Errorf("unsupported message type: %T", v)
}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
	dst, ok := v.(*[]byte)
	if ok {
		*dst = data
		return nil
	}
	pt, ok := v.(proto.Message)
	if ok {
		return proto.Unmarshal(data, pt)
	}
	return fmt.Errorf("unsupported message type: %T", v)
}

func (c *rawCodec) String() string {
	return fmt.Sprintf("proxy codec")
}

//
//type protoCodec struct{}
//
//func (protoCodec) Marshal(v interface{}) ([]byte, error) {
//	return proto.Marshal(v.(proto.Message))
//}
//
//func (protoCodec) Unmarshal(data []byte, v interface{}) error {
//	return proto.Unmarshal(data, v.(proto.Message))
//}
//
//func (protoCodec) String() string {
//	return "proto"
//}
