// Code generated by protoc-gen-go.
// source: admin.proto
// DO NOT EDIT!

/*
Package logdog is a generated protocol buffer package.

It is generated from these files:
	admin.proto

It has these top-level messages:
	SetConfigRequest
*/
package logdog

import prpccommon "github.com/luci/luci-go/common/prpc"
import prpc "github.com/luci/luci-go/server/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/luci/luci-go/common/proto/google"

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

// GlobalConfig is the LogDog Coordinator global configuration.
//
// This is intended to act as an entry point. The majority of the configuration
// will be stored in a "luci-config" service Config protobuf.
type SetConfigRequest struct {
	// ConfigServiceURL is the API URL of the base "luci-config" service. If
	// empty, the defualt service URL will be used.
	ConfigServiceUrl string `protobuf:"bytes,1,opt,name=config_service_url,json=configServiceUrl" json:"config_service_url,omitempty"`
	// ConfigSet is the name of the configuration set to load from.
	ConfigSet string `protobuf:"bytes,2,opt,name=config_set,json=configSet" json:"config_set,omitempty"`
	// ConfigPath is the path of the text-serialized configuration protobuf.
	ConfigPath string `protobuf:"bytes,3,opt,name=config_path,json=configPath" json:"config_path,omitempty"`
	// If not empty, is the service account JSON file data that will be used for
	// Storage access.
	//
	// TODO(dnj): Remove this option once Cloud BigTable has cross-project ACLs.
	StorageServiceAccountJson []byte `protobuf:"bytes,100,opt,name=storage_service_account_json,json=storageServiceAccountJson,proto3" json:"storage_service_account_json,omitempty"`
}

func (m *SetConfigRequest) Reset()                    { *m = SetConfigRequest{} }
func (m *SetConfigRequest) String() string            { return proto.CompactTextString(m) }
func (*SetConfigRequest) ProtoMessage()               {}
func (*SetConfigRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func init() {
	proto.RegisterType((*SetConfigRequest)(nil), "logdog.SetConfigRequest")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion2

// Client API for Admin service

type AdminClient interface {
	// SetConfig loads the supplied configuration into a config.GlobalConfig
	// instance.
	SetConfig(ctx context.Context, in *SetConfigRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
}
type adminPRPCClient struct {
	client *prpccommon.Client
}

func NewAdminPRPCClient(client *prpccommon.Client) AdminClient {
	return &adminPRPCClient{client}
}

func (c *adminPRPCClient) SetConfig(ctx context.Context, in *SetConfigRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	out := new(google_protobuf.Empty)
	err := c.client.Call(ctx, "logdog.Admin", "SetConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type adminClient struct {
	cc *grpc.ClientConn
}

func NewAdminClient(cc *grpc.ClientConn) AdminClient {
	return &adminClient{cc}
}

func (c *adminClient) SetConfig(ctx context.Context, in *SetConfigRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error) {
	out := new(google_protobuf.Empty)
	err := grpc.Invoke(ctx, "/logdog.Admin/SetConfig", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Admin service

type AdminServer interface {
	// SetConfig loads the supplied configuration into a config.GlobalConfig
	// instance.
	SetConfig(context.Context, *SetConfigRequest) (*google_protobuf.Empty, error)
}

func RegisterAdminServer(s prpc.Registrar, srv AdminServer) {
	s.RegisterService(&_Admin_serviceDesc, srv)
}

func _Admin_SetConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AdminServer).SetConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logdog.Admin/SetConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AdminServer).SetConfig(ctx, req.(*SetConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Admin_serviceDesc = grpc.ServiceDesc{
	ServiceName: "logdog.Admin",
	HandlerType: (*AdminServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetConfig",
			Handler:    _Admin_SetConfig_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 237 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x5c, 0x90, 0x41, 0x4b, 0xc3, 0x40,
	0x10, 0x85, 0x89, 0x62, 0x21, 0x53, 0x0f, 0x65, 0x0e, 0xb2, 0x56, 0x45, 0xf1, 0xe4, 0x41, 0xb6,
	0xa0, 0x67, 0x91, 0x22, 0x7a, 0xf0, 0x24, 0x29, 0x9e, 0x43, 0x9a, 0x4c, 0xd7, 0x48, 0x9a, 0x89,
	0x9b, 0x59, 0xc1, 0x9f, 0xe7, 0x3f, 0x33, 0xdd, 0xdd, 0xe6, 0xe0, 0x71, 0xde, 0xfb, 0x86, 0xc7,
	0x7b, 0x30, 0x2d, 0xaa, 0x6d, 0xdd, 0xea, 0xce, 0xb2, 0x30, 0x4e, 0x1a, 0x36, 0x15, 0x9b, 0xf9,
	0x99, 0x61, 0x36, 0x0d, 0x2d, 0xbc, 0xba, 0x76, 0x9b, 0x05, 0x6d, 0x3b, 0xf9, 0x09, 0xd0, 0xf5,
	0x6f, 0x02, 0xb3, 0x15, 0xc9, 0x13, 0xb7, 0x9b, 0xda, 0x64, 0xf4, 0xe5, 0xa8, 0x17, 0xbc, 0x05,
	0x2c, 0xbd, 0x90, 0xf7, 0x64, 0xbf, 0xeb, 0x92, 0x72, 0x67, 0x1b, 0x95, 0x5c, 0x25, 0x37, 0x69,
	0x36, 0x0b, 0xce, 0x2a, 0x18, 0xef, 0xb6, 0xc1, 0x0b, 0x80, 0x91, 0x16, 0x75, 0xe0, 0xa9, 0x74,
	0x4f, 0x09, 0x5e, 0xc2, 0x34, 0xda, 0x5d, 0x21, 0x1f, 0xea, 0xd0, 0xfb, 0xf1, 0xe3, 0x6d, 0x50,
	0xf0, 0x11, 0xce, 0x7b, 0x61, 0x5b, 0x18, 0x1a, 0xe3, 0x8a, 0xb2, 0x64, 0xd7, 0x4a, 0xfe, 0xd9,
	0x73, 0xab, 0xaa, 0xe1, 0xe3, 0x38, 0x3b, 0x8d, 0x4c, 0x0c, 0x5e, 0x06, 0xe2, 0x75, 0x00, 0xee,
	0x5e, 0xe0, 0x68, 0xb9, 0xeb, 0x8d, 0x0f, 0x90, 0x8e, 0x5d, 0x50, 0xe9, 0xd0, 0x5f, 0xff, 0xaf,
	0x37, 0x3f, 0xd1, 0x61, 0x11, 0xbd, 0x5f, 0x44, 0x3f, 0xef, 0x16, 0x59, 0x4f, 0xfc, 0x7d, 0xff,
	0x17, 0x00, 0x00, 0xff, 0xff, 0xb3, 0xb4, 0x6c, 0x90, 0x46, 0x01, 0x00, 0x00,
}
