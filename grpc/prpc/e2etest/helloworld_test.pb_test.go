// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/luci/luci-go/grpc/prpc/e2etest/helloworld_test.proto

/*
Package e2etest is a generated protocol buffer package.

It is generated from these files:
	github.com/luci/luci-go/grpc/prpc/e2etest/helloworld_test.proto

It has these top-level messages:
	HelloRequest
	HelloReply
*/
package e2etest

import prpc "github.com/luci/luci-go/grpc/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// The request message containing the user's name.
type HelloRequest struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *HelloRequest) Reset()                    { *m = HelloRequest{} }
func (m *HelloRequest) String() string            { return proto.CompactTextString(m) }
func (*HelloRequest) ProtoMessage()               {}
func (*HelloRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *HelloRequest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

// The response message containing the greetings
type HelloReply struct {
	Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
}

func (m *HelloReply) Reset()                    { *m = HelloReply{} }
func (m *HelloReply) String() string            { return proto.CompactTextString(m) }
func (*HelloReply) ProtoMessage()               {}
func (*HelloReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *HelloReply) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterType((*HelloRequest)(nil), "e2etest.HelloRequest")
	proto.RegisterType((*HelloReply)(nil), "e2etest.HelloReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Hello service

type HelloClient interface {
	Greet(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error)
}
type helloPRPCClient struct {
	client *prpc.Client
}

func NewHelloPRPCClient(client *prpc.Client) HelloClient {
	return &helloPRPCClient{client}
}

func (c *helloPRPCClient) Greet(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	out := new(HelloReply)
	err := c.client.Call(ctx, "e2etest.Hello", "Greet", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type helloClient struct {
	cc *grpc.ClientConn
}

func NewHelloClient(cc *grpc.ClientConn) HelloClient {
	return &helloClient{cc}
}

func (c *helloClient) Greet(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	out := new(HelloReply)
	err := grpc.Invoke(ctx, "/e2etest.Hello/Greet", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Hello service

type HelloServer interface {
	Greet(context.Context, *HelloRequest) (*HelloReply, error)
}

func RegisterHelloServer(s prpc.Registrar, srv HelloServer) {
	s.RegisterService(&_Hello_serviceDesc, srv)
}

func _Hello_Greet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HelloServer).Greet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/e2etest.Hello/Greet",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HelloServer).Greet(ctx, req.(*HelloRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Hello_serviceDesc = grpc.ServiceDesc{
	ServiceName: "e2etest.Hello",
	HandlerType: (*HelloServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Greet",
			Handler:    _Hello_Greet_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/luci/luci-go/grpc/prpc/e2etest/helloworld_test.proto",
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/grpc/prpc/e2etest/helloworld_test.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 174 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xb2, 0x4f, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xcf, 0x29, 0x4d, 0xce, 0x04, 0x13, 0xba, 0xe9, 0xf9,
	0xfa, 0xe9, 0x45, 0x05, 0xc9, 0xfa, 0x05, 0x20, 0x22, 0xd5, 0x28, 0xb5, 0x24, 0xb5, 0xb8, 0x44,
	0x3f, 0x23, 0x35, 0x27, 0x27, 0xbf, 0x3c, 0xbf, 0x28, 0x27, 0x25, 0x1e, 0xc4, 0xd7, 0x2b, 0x28,
	0xca, 0x2f, 0xc9, 0x17, 0x62, 0x87, 0x4a, 0x2b, 0x29, 0x71, 0xf1, 0x78, 0x80, 0x54, 0x04, 0xa5,
	0x16, 0x96, 0xa6, 0x16, 0x97, 0x08, 0x09, 0x71, 0xb1, 0xe4, 0x25, 0xe6, 0xa6, 0x4a, 0x30, 0x2a,
	0x30, 0x6a, 0x70, 0x06, 0x81, 0xd9, 0x4a, 0x6a, 0x5c, 0x5c, 0x50, 0x35, 0x05, 0x39, 0x95, 0x42,
	0x12, 0x5c, 0xec, 0xb9, 0xa9, 0xc5, 0xc5, 0x89, 0xe9, 0x30, 0x45, 0x30, 0xae, 0x91, 0x0d, 0x17,
	0x2b, 0x58, 0x9d, 0x90, 0x31, 0x17, 0xab, 0x7b, 0x51, 0x6a, 0x6a, 0x89, 0x90, 0xa8, 0x1e, 0xd4,
	0x1e, 0x3d, 0x64, 0x4b, 0xa4, 0x84, 0xd1, 0x85, 0x0b, 0x72, 0x2a, 0x93, 0xd8, 0xc0, 0x2e, 0x33,
	0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x4a, 0x4b, 0xe8, 0xc3, 0xdc, 0x00, 0x00, 0x00,
}
