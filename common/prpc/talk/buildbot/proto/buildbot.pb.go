// Code generated by protoc-gen-go.
// source: buildbot.proto
// DO NOT EDIT!

/*
Package buildbot is a generated protocol buffer package.

It is generated from these files:
	buildbot.proto

It has these top-level messages:
	SearchRequest
	SearchResponse
	Build
	ScheduleRequest
	ScheduleResponse
*/
package buildbot

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

type BuildState int32

const (
	BuildState_UNSET     BuildState = 0
	BuildState_PENDING   BuildState = 1
	BuildState_RUNNING   BuildState = 2
	BuildState_SUCCESS   BuildState = 3
	BuildState_FAILURE   BuildState = 4
	BuildState_EXCEPTION BuildState = 5
)

var BuildState_name = map[int32]string{
	0: "UNSET",
	1: "PENDING",
	2: "RUNNING",
	3: "SUCCESS",
	4: "FAILURE",
	5: "EXCEPTION",
}
var BuildState_value = map[string]int32{
	"UNSET":     0,
	"PENDING":   1,
	"RUNNING":   2,
	"SUCCESS":   3,
	"FAILURE":   4,
	"EXCEPTION": 5,
}

func (x BuildState) String() string {
	return proto.EnumName(BuildState_name, int32(x))
}
func (BuildState) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// SearchReqeust specifies a search criteria.
type SearchRequest struct {
	// Master filters by master name, e.g. "master.XXX".
	Master string `protobuf:"bytes,1,opt,name=master" json:"master,omitempty"`
	// State filters by build state.
	State BuildState `protobuf:"varint,2,opt,name=state,enum=buildbot.BuildState" json:"state,omitempty"`
	// Builder filters by builder name.
	Builder string `protobuf:"bytes,3,opt,name=builder" json:"builder,omitempty"`
}

func (m *SearchRequest) Reset()                    { *m = SearchRequest{} }
func (m *SearchRequest) String() string            { return proto.CompactTextString(m) }
func (*SearchRequest) ProtoMessage()               {}
func (*SearchRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type SearchResponse struct {
	Builds []*Build `protobuf:"bytes,1,rep,name=builds" json:"builds,omitempty"`
}

func (m *SearchResponse) Reset()                    { *m = SearchResponse{} }
func (m *SearchResponse) String() string            { return proto.CompactTextString(m) }
func (*SearchResponse) ProtoMessage()               {}
func (*SearchResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *SearchResponse) GetBuilds() []*Build {
	if m != nil {
		return m.Builds
	}
	return nil
}

type Build struct {
	Master  string     `protobuf:"bytes,1,opt,name=master" json:"master,omitempty"`
	Builder string     `protobuf:"bytes,2,opt,name=builder" json:"builder,omitempty"`
	Number  int32      `protobuf:"varint,3,opt,name=number" json:"number,omitempty"`
	State   BuildState `protobuf:"varint,4,opt,name=state,enum=buildbot.BuildState" json:"state,omitempty"`
}

func (m *Build) Reset()                    { *m = Build{} }
func (m *Build) String() string            { return proto.CompactTextString(m) }
func (*Build) ProtoMessage()               {}
func (*Build) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

// ScheduleRequest defines builds to schedule.
type ScheduleRequest struct {
	// Master is a "master.XXX" string that defines where to schedule builds.
	Master string `protobuf:"bytes,1,opt,name=master" json:"master,omitempty"`
	// Builds is a list of builds to schedule.
	Builds []*ScheduleRequest_BuildDef `protobuf:"bytes,2,rep,name=builds" json:"builds,omitempty"`
}

func (m *ScheduleRequest) Reset()                    { *m = ScheduleRequest{} }
func (m *ScheduleRequest) String() string            { return proto.CompactTextString(m) }
func (*ScheduleRequest) ProtoMessage()               {}
func (*ScheduleRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *ScheduleRequest) GetBuilds() []*ScheduleRequest_BuildDef {
	if m != nil {
		return m.Builds
	}
	return nil
}

// Build is a build to schedule.
type ScheduleRequest_BuildDef struct {
	// Builder defines the build script.
	Builder string `protobuf:"bytes,1,opt,name=builder" json:"builder,omitempty"`
	// Branch defines what to fetch.
	Branch string `protobuf:"bytes,2,opt,name=branch" json:"branch,omitempty"`
	// Revision is a commit hash to checkout
	Revision string `protobuf:"bytes,3,opt,name=revision" json:"revision,omitempty"`
	// Properties are "key:value" pairs.
	Properties []string `protobuf:"bytes,4,rep,name=properties" json:"properties,omitempty"`
	// Blamelist is a list of user email addressed to blame if this build
	// fails.
	Blamelist []string `protobuf:"bytes,5,rep,name=blamelist" json:"blamelist,omitempty"`
}

func (m *ScheduleRequest_BuildDef) Reset()                    { *m = ScheduleRequest_BuildDef{} }
func (m *ScheduleRequest_BuildDef) String() string            { return proto.CompactTextString(m) }
func (*ScheduleRequest_BuildDef) ProtoMessage()               {}
func (*ScheduleRequest_BuildDef) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0} }

// HelloReply contains a greeting.
type ScheduleResponse struct {
	Builds []*Build `protobuf:"bytes,1,rep,name=builds" json:"builds,omitempty"`
}

func (m *ScheduleResponse) Reset()                    { *m = ScheduleResponse{} }
func (m *ScheduleResponse) String() string            { return proto.CompactTextString(m) }
func (*ScheduleResponse) ProtoMessage()               {}
func (*ScheduleResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *ScheduleResponse) GetBuilds() []*Build {
	if m != nil {
		return m.Builds
	}
	return nil
}

func init() {
	proto.RegisterType((*SearchRequest)(nil), "buildbot.SearchRequest")
	proto.RegisterType((*SearchResponse)(nil), "buildbot.SearchResponse")
	proto.RegisterType((*Build)(nil), "buildbot.Build")
	proto.RegisterType((*ScheduleRequest)(nil), "buildbot.ScheduleRequest")
	proto.RegisterType((*ScheduleRequest_BuildDef)(nil), "buildbot.ScheduleRequest.BuildDef")
	proto.RegisterType((*ScheduleResponse)(nil), "buildbot.ScheduleResponse")
	proto.RegisterEnum("buildbot.BuildState", BuildState_name, BuildState_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion2

// Client API for Buildbot service

type BuildbotClient interface {
	// Search returns builds matching a criteria.
	Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error)
	// Schedule puts new builds to a queue.
	Schedule(ctx context.Context, in *ScheduleRequest, opts ...grpc.CallOption) (*ScheduleResponse, error)
}
type buildbotPRPCClient struct {
	client *prpccommon.Client
}

func NewBuildbotPRPCClient(client *prpccommon.Client) BuildbotClient {
	return &buildbotPRPCClient{client}
}

func (c *buildbotPRPCClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	out := new(SearchResponse)
	err := c.client.Call(ctx, "buildbot.Buildbot", "Search", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *buildbotPRPCClient) Schedule(ctx context.Context, in *ScheduleRequest, opts ...grpc.CallOption) (*ScheduleResponse, error) {
	out := new(ScheduleResponse)
	err := c.client.Call(ctx, "buildbot.Buildbot", "Schedule", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type buildbotClient struct {
	cc *grpc.ClientConn
}

func NewBuildbotClient(cc *grpc.ClientConn) BuildbotClient {
	return &buildbotClient{cc}
}

func (c *buildbotClient) Search(ctx context.Context, in *SearchRequest, opts ...grpc.CallOption) (*SearchResponse, error) {
	out := new(SearchResponse)
	err := grpc.Invoke(ctx, "/buildbot.Buildbot/Search", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *buildbotClient) Schedule(ctx context.Context, in *ScheduleRequest, opts ...grpc.CallOption) (*ScheduleResponse, error) {
	out := new(ScheduleResponse)
	err := grpc.Invoke(ctx, "/buildbot.Buildbot/Schedule", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Buildbot service

type BuildbotServer interface {
	// Search returns builds matching a criteria.
	Search(context.Context, *SearchRequest) (*SearchResponse, error)
	// Schedule puts new builds to a queue.
	Schedule(context.Context, *ScheduleRequest) (*ScheduleResponse, error)
}

func RegisterBuildbotServer(s prpc.Registrar, srv BuildbotServer) {
	s.RegisterService(&_Buildbot_serviceDesc, srv)
}

func _Buildbot_Search_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildbotServer).Search(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buildbot.Buildbot/Search",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildbotServer).Search(ctx, req.(*SearchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Buildbot_Schedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildbotServer).Schedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/buildbot.Buildbot/Schedule",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildbotServer).Schedule(ctx, req.(*ScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Buildbot_serviceDesc = grpc.ServiceDesc{
	ServiceName: "buildbot.Buildbot",
	HandlerType: (*BuildbotServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Search",
			Handler:    _Buildbot_Search_Handler,
		},
		{
			MethodName: "Schedule",
			Handler:    _Buildbot_Schedule_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 416 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x94, 0x93, 0x4f, 0x8f, 0x93, 0x50,
	0x14, 0xc5, 0x0b, 0x14, 0x0a, 0xb7, 0x69, 0x4b, 0x5e, 0x8c, 0x22, 0x31, 0xc6, 0xb0, 0xb1, 0xe9,
	0xa2, 0x8b, 0xba, 0x52, 0xe3, 0x42, 0x29, 0x9a, 0x26, 0x06, 0x1b, 0x28, 0x89, 0x71, 0x07, 0xed,
	0x33, 0x25, 0xe1, 0x4f, 0xe5, 0x3d, 0xdc, 0xcd, 0x57, 0x98, 0xe5, 0x7c, 0xdc, 0xc9, 0xf0, 0xef,
	0x15, 0xa6, 0x9d, 0xa6, 0x99, 0x1d, 0xe7, 0x1e, 0x38, 0xf7, 0xfd, 0xee, 0x7d, 0xc0, 0x38, 0xc8,
	0xc3, 0x68, 0x17, 0xa4, 0x74, 0x7e, 0xc8, 0x52, 0x9a, 0x22, 0x99, 0x69, 0x23, 0x86, 0x91, 0x8b,
	0xfd, 0x6c, 0xbb, 0x77, 0xf0, 0xbf, 0x1c, 0x13, 0x8a, 0x5e, 0x82, 0x14, 0xfb, 0x84, 0xe2, 0x4c,
	0xe3, 0xde, 0x71, 0x53, 0xc5, 0x69, 0x14, 0x9a, 0x81, 0x48, 0xa8, 0x4f, 0xb1, 0xc6, 0x17, 0xe5,
	0xf1, 0xe2, 0xc5, 0xfc, 0x18, 0xf9, 0xad, 0x7c, 0x70, 0x4b, 0xcf, 0xa9, 0x5f, 0x41, 0x1a, 0x0c,
	0x2a, 0xb7, 0x08, 0x11, 0xaa, 0x10, 0x26, 0x8d, 0x8f, 0x30, 0x66, 0xed, 0xc8, 0x21, 0x4d, 0x08,
	0x46, 0xef, 0x41, 0xaa, 0x4c, 0x52, 0xf4, 0x13, 0xa6, 0xc3, 0xc5, 0xe4, 0x24, 0xd8, 0x69, 0x6c,
	0xe3, 0x06, 0xc4, 0xaa, 0x70, 0xf1, 0x84, 0x9d, 0xae, 0xfc, 0xa3, 0xae, 0xe5, 0x17, 0x49, 0x1e,
	0x07, 0xcd, 0x71, 0x44, 0xa7, 0x51, 0x2d, 0x53, 0xff, 0x2a, 0x93, 0x71, 0xcf, 0xc1, 0xc4, 0xdd,
	0xee, 0xf1, 0x2e, 0x8f, 0xf0, 0xb5, 0x59, 0x7d, 0x3a, 0x32, 0xf1, 0x15, 0x93, 0xd1, 0x06, 0x9f,
	0x44, 0xd4, 0x8d, 0x96, 0xf8, 0x2f, 0xc3, 0xd4, 0xef, 0x38, 0x90, 0x59, 0xb1, 0x8b, 0xc4, 0x9d,
	0x21, 0x05, 0x99, 0x9f, 0x6c, 0xf7, 0x0d, 0x6b, 0xa3, 0x90, 0x0e, 0x72, 0x86, 0xff, 0x87, 0x24,
	0x4c, 0x93, 0x66, 0xf6, 0x47, 0x8d, 0xde, 0x02, 0x14, 0xeb, 0x3f, 0xe0, 0x8c, 0x86, 0x98, 0x14,
	0xcc, 0x42, 0xe1, 0x76, 0x2a, 0xe8, 0x0d, 0x28, 0x41, 0xe4, 0xc7, 0x38, 0x0a, 0x09, 0xd5, 0xc4,
	0xca, 0x6e, 0x0b, 0xc6, 0x67, 0x50, 0xdb, 0xc3, 0x3f, 0x73, 0x79, 0xb3, 0x3f, 0x00, 0xed, 0x48,
	0x91, 0x02, 0xa2, 0x67, 0xbb, 0xd6, 0x46, 0xed, 0xa1, 0x21, 0x0c, 0xd6, 0x96, 0xbd, 0x5c, 0xd9,
	0x3f, 0x54, 0xae, 0x14, 0x8e, 0x67, 0xdb, 0xa5, 0xe0, 0x4b, 0xe1, 0x7a, 0xa6, 0x69, 0xb9, 0xae,
	0x2a, 0x94, 0xe2, 0xfb, 0xd7, 0xd5, 0x4f, 0xcf, 0xb1, 0xd4, 0x3e, 0x1a, 0x81, 0x62, 0xfd, 0x36,
	0xad, 0xf5, 0x66, 0xf5, 0xcb, 0x56, 0xc5, 0xc5, 0x2d, 0x9b, 0x58, 0xd1, 0x16, 0x7d, 0x01, 0xa9,
	0xbe, 0x60, 0xe8, 0x55, 0x67, 0xe8, 0xdd, 0x1b, 0xae, 0x6b, 0xe7, 0x46, 0x8d, 0x63, 0xf4, 0x90,
	0x09, 0x32, 0x83, 0x44, 0xaf, 0x2f, 0x6e, 0x4d, 0xd7, 0x9f, 0xb2, 0x58, 0x48, 0x20, 0x55, 0x3f,
	0xd9, 0x87, 0x87, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb6, 0x7a, 0x7d, 0x17, 0x76, 0x03, 0x00, 0x00,
}
