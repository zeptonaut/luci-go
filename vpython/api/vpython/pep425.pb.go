// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/luci/luci-go/vpython/api/vpython/pep425.proto

package vpython

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Represents a Python PEP425 tag.
type PEP425Tag struct {
	// Python is the PEP425 python tag (e.g., "cp27").
	Python string `protobuf:"bytes,1,opt,name=python" json:"python,omitempty"`
	// ABI is the PEP425 "python ABI" tag (e.g., "cp27mu", "none").
	Abi string `protobuf:"bytes,2,opt,name=abi" json:"abi,omitempty"`
	// Platform is the PEP425 "python platform" tag (e.g., "linux_x86_64",
	// "armv7l", "any").
	Platform string `protobuf:"bytes,3,opt,name=platform" json:"platform,omitempty"`
}

func (m *PEP425Tag) Reset()                    { *m = PEP425Tag{} }
func (m *PEP425Tag) String() string            { return proto.CompactTextString(m) }
func (*PEP425Tag) ProtoMessage()               {}
func (*PEP425Tag) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *PEP425Tag) GetPython() string {
	if m != nil {
		return m.Python
	}
	return ""
}

func (m *PEP425Tag) GetAbi() string {
	if m != nil {
		return m.Abi
	}
	return ""
}

func (m *PEP425Tag) GetPlatform() string {
	if m != nil {
		return m.Platform
	}
	return ""
}

func init() {
	proto.RegisterType((*PEP425Tag)(nil), "vpython.PEP425Tag")
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/vpython/api/vpython/pep425.proto", fileDescriptor1)
}

var fileDescriptor1 = []byte{
	// 138 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xb2, 0x48, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xcf, 0x29, 0x4d, 0xce, 0x04, 0x13, 0xba, 0xe9, 0xf9,
	0xfa, 0x65, 0x05, 0x95, 0x25, 0x19, 0xf9, 0x79, 0xfa, 0x89, 0x05, 0x99, 0x70, 0x76, 0x41, 0x6a,
	0x81, 0x89, 0x91, 0xa9, 0x5e, 0x41, 0x51, 0x7e, 0x49, 0xbe, 0x10, 0x3b, 0x54, 0x54, 0x29, 0x90,
	0x8b, 0x33, 0xc0, 0x35, 0xc0, 0xc4, 0xc8, 0x34, 0x24, 0x31, 0x5d, 0x48, 0x8c, 0x8b, 0x0d, 0x22,
	0x2c, 0xc1, 0xa8, 0xc0, 0xa8, 0xc1, 0x19, 0x04, 0xe5, 0x09, 0x09, 0x70, 0x31, 0x27, 0x26, 0x65,
	0x4a, 0x30, 0x81, 0x05, 0x41, 0x4c, 0x21, 0x29, 0x2e, 0x8e, 0x82, 0x9c, 0xc4, 0x92, 0xb4, 0xfc,
	0xa2, 0x5c, 0x09, 0x66, 0xb0, 0x30, 0x9c, 0x9f, 0xc4, 0x06, 0xb6, 0xc2, 0x18, 0x10, 0x00, 0x00,
	0xff, 0xff, 0x54, 0x7c, 0x40, 0x1b, 0x9e, 0x00, 0x00, 0x00,
}
