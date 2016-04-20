// Code generated by protoc-gen-go.
// source: helloworld.proto
// DO NOT EDIT!

/*
Package testservices is a generated protocol buffer package.

It is generated from these files:
	helloworld.proto

It has these top-level messages:
	HelloRequest
	HelloReply
	MultiplyRequest
	MultiplyResponse
*/
package testservices

import prpccommon "github.com/luci/luci-go/common/prpc"
import prpc "github.com/luci/luci-go/server/prpc"

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
const _ = proto.ProtoPackageIsVersion1

// The request message containing the user's name.
type HelloRequest struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *HelloRequest) Reset()                    { *m = HelloRequest{} }
func (m *HelloRequest) String() string            { return proto.CompactTextString(m) }
func (*HelloRequest) ProtoMessage()               {}
func (*HelloRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// The response message containing the greetings
type HelloReply struct {
	Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
}

func (m *HelloReply) Reset()                    { *m = HelloReply{} }
func (m *HelloReply) String() string            { return proto.CompactTextString(m) }
func (*HelloReply) ProtoMessage()               {}
func (*HelloReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type MultiplyRequest struct {
	X int32 `protobuf:"varint,1,opt,name=x" json:"x,omitempty"`
	Y int32 `protobuf:"varint,2,opt,name=y" json:"y,omitempty"`
}

func (m *MultiplyRequest) Reset()                    { *m = MultiplyRequest{} }
func (m *MultiplyRequest) String() string            { return proto.CompactTextString(m) }
func (*MultiplyRequest) ProtoMessage()               {}
func (*MultiplyRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type MultiplyResponse struct {
	Z int32 `protobuf:"varint,1,opt,name=z" json:"z,omitempty"`
}

func (m *MultiplyResponse) Reset()                    { *m = MultiplyResponse{} }
func (m *MultiplyResponse) String() string            { return proto.CompactTextString(m) }
func (*MultiplyResponse) ProtoMessage()               {}
func (*MultiplyResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func init() {
	proto.RegisterType((*HelloRequest)(nil), "testservices.HelloRequest")
	proto.RegisterType((*HelloReply)(nil), "testservices.HelloReply")
	proto.RegisterType((*MultiplyRequest)(nil), "testservices.MultiplyRequest")
	proto.RegisterType((*MultiplyResponse)(nil), "testservices.MultiplyResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion2

// Client API for Greeter service

type GreeterClient interface {
	// Sends a greeting
	SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error)
}
type greeterPRPCClient struct {
	client *prpccommon.Client
}

func NewGreeterPRPCClient(client *prpccommon.Client) GreeterClient {
	return &greeterPRPCClient{client}
}

func (c *greeterPRPCClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	out := new(HelloReply)
	err := c.client.Call(ctx, "testservices.Greeter", "SayHello", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type greeterClient struct {
	cc *grpc.ClientConn
}

func NewGreeterClient(cc *grpc.ClientConn) GreeterClient {
	return &greeterClient{cc}
}

func (c *greeterClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
	out := new(HelloReply)
	err := grpc.Invoke(ctx, "/testservices.Greeter/SayHello", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Greeter service

type GreeterServer interface {
	// Sends a greeting
	SayHello(context.Context, *HelloRequest) (*HelloReply, error)
}

func RegisterGreeterServer(s prpc.Registrar, srv GreeterServer) {
	s.RegisterService(&_Greeter_serviceDesc, srv)
}

func _Greeter_SayHello_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HelloRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GreeterServer).SayHello(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/testservices.Greeter/SayHello",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GreeterServer).SayHello(ctx, req.(*HelloRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Greeter_serviceDesc = grpc.ServiceDesc{
	ServiceName: "testservices.Greeter",
	HandlerType: (*GreeterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Greeter_SayHello_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

// Client API for Calc service

type CalcClient interface {
	Multiply(ctx context.Context, in *MultiplyRequest, opts ...grpc.CallOption) (*MultiplyResponse, error)
}
type calcPRPCClient struct {
	client *prpccommon.Client
}

func NewCalcPRPCClient(client *prpccommon.Client) CalcClient {
	return &calcPRPCClient{client}
}

func (c *calcPRPCClient) Multiply(ctx context.Context, in *MultiplyRequest, opts ...grpc.CallOption) (*MultiplyResponse, error) {
	out := new(MultiplyResponse)
	err := c.client.Call(ctx, "testservices.Calc", "Multiply", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type calcClient struct {
	cc *grpc.ClientConn
}

func NewCalcClient(cc *grpc.ClientConn) CalcClient {
	return &calcClient{cc}
}

func (c *calcClient) Multiply(ctx context.Context, in *MultiplyRequest, opts ...grpc.CallOption) (*MultiplyResponse, error) {
	out := new(MultiplyResponse)
	err := grpc.Invoke(ctx, "/testservices.Calc/Multiply", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Calc service

type CalcServer interface {
	Multiply(context.Context, *MultiplyRequest) (*MultiplyResponse, error)
}

func RegisterCalcServer(s prpc.Registrar, srv CalcServer) {
	s.RegisterService(&_Calc_serviceDesc, srv)
}

func _Calc_Multiply_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MultiplyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CalcServer).Multiply(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/testservices.Calc/Multiply",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CalcServer).Multiply(ctx, req.(*MultiplyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Calc_serviceDesc = grpc.ServiceDesc{
	ServiceName: "testservices.Calc",
	HandlerType: (*CalcServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Multiply",
			Handler:    _Calc_Multiply_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 228 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x74, 0xd0, 0xc1, 0x4a, 0xc4, 0x30,
	0x10, 0x06, 0xe0, 0x8d, 0xac, 0xee, 0x3a, 0x14, 0x5c, 0xe6, 0x14, 0x0a, 0xca, 0x92, 0x83, 0x78,
	0xb1, 0x87, 0xf5, 0x0d, 0xf4, 0xa0, 0x20, 0x7b, 0xe9, 0x3e, 0x41, 0xac, 0x83, 0x16, 0xd2, 0xa6,
	0x26, 0xa9, 0x36, 0x7d, 0x7a, 0xdb, 0xd4, 0x60, 0x15, 0xbd, 0xf5, 0xef, 0x7c, 0x64, 0xfe, 0x04,
	0x36, 0xaf, 0xa4, 0x94, 0xfe, 0xd0, 0x46, 0x3d, 0x67, 0x8d, 0xd1, 0x4e, 0x63, 0xe2, 0xc8, 0x3a,
	0x4b, 0xe6, 0xbd, 0x2c, 0xc8, 0x0a, 0x01, 0xc9, 0xc3, 0x28, 0x72, 0x7a, 0x6b, 0x87, 0xff, 0x88,
	0xb0, 0xac, 0x65, 0x45, 0x9c, 0x6d, 0xd9, 0xd5, 0x69, 0x1e, 0xbe, 0xc5, 0x25, 0xc0, 0x97, 0x69,
	0x94, 0x47, 0x0e, 0xab, 0x8a, 0xac, 0x95, 0x2f, 0x11, 0xc5, 0x28, 0xae, 0xe1, 0x6c, 0xdf, 0x2a,
	0x57, 0x0e, 0x2a, 0x1e, 0x97, 0x00, 0xeb, 0x02, 0x3b, 0xce, 0x59, 0x37, 0x26, 0xcf, 0x8f, 0xa6,
	0xe4, 0xc5, 0x16, 0x36, 0xdf, 0xdc, 0x36, 0xba, 0xb6, 0x34, 0x8a, 0x3e, 0xfa, 0x7e, 0xb7, 0x87,
	0xd5, 0xbd, 0x21, 0x72, 0x64, 0xf0, 0x16, 0xd6, 0x07, 0xe9, 0x43, 0x0d, 0x4c, 0xb3, 0xf9, 0x15,
	0xb2, 0x79, 0xff, 0x94, 0xff, 0x39, 0x1b, 0x56, 0x88, 0xc5, 0xee, 0x00, 0xcb, 0x3b, 0xa9, 0x0a,
	0x7c, 0x84, 0x75, 0x5c, 0x8c, 0xe7, 0x3f, 0xfd, 0xaf, 0xfe, 0xe9, 0xc5, 0x7f, 0xe3, 0xa9, 0xaf,
	0x58, 0x3c, 0x9d, 0x84, 0x57, 0xbd, 0xf9, 0x0c, 0x00, 0x00, 0xff, 0xff, 0x89, 0x2d, 0xce, 0x57,
	0x69, 0x01, 0x00, 0x00,
}
