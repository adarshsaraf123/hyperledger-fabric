// Code generated by protoc-gen-go. DO NOT EDIT.
// source: orderer/etcdraft/configuration.proto

package etcdraft // import "github.com/hyperledger/fabric/protos/orderer/etcdraft"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Metadata is serialized and set as the value of ConsensusType.Metadata in
// a channel configuration when the ConsensusType.Type is set "etcdraft".
type Metadata struct {
	Consenters           []*Consenter `protobuf:"bytes,1,rep,name=consenters" json:"consenters,omitempty"`
	FSMConfig            *FSMConfig   `protobuf:"bytes,2,opt,name=FSM_config,json=FSMConfig" json:"FSM_config,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *Metadata) Reset()         { *m = Metadata{} }
func (m *Metadata) String() string { return proto.CompactTextString(m) }
func (*Metadata) ProtoMessage()    {}
func (*Metadata) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_5df0b1cd4230b47f, []int{0}
}
func (m *Metadata) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Metadata.Unmarshal(m, b)
}
func (m *Metadata) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Metadata.Marshal(b, m, deterministic)
}
func (dst *Metadata) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Metadata.Merge(dst, src)
}
func (m *Metadata) XXX_Size() int {
	return xxx_messageInfo_Metadata.Size(m)
}
func (m *Metadata) XXX_DiscardUnknown() {
	xxx_messageInfo_Metadata.DiscardUnknown(m)
}

var xxx_messageInfo_Metadata proto.InternalMessageInfo

func (m *Metadata) GetConsenters() []*Consenter {
	if m != nil {
		return m.Consenters
	}
	return nil
}

func (m *Metadata) GetFSMConfig() *FSMConfig {
	if m != nil {
		return m.FSMConfig
	}
	return nil
}

// Consenter represents a consenting node (i.e. replica).
type Consenter struct {
	Host                 string   `protobuf:"bytes,1,opt,name=host" json:"host,omitempty"`
	Port                 uint32   `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
	ClientTlsCert        []byte   `protobuf:"bytes,3,opt,name=client_tls_cert,json=clientTlsCert,proto3" json:"client_tls_cert,omitempty"`
	ServerTlsCert        []byte   `protobuf:"bytes,4,opt,name=server_tls_cert,json=serverTlsCert,proto3" json:"server_tls_cert,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Consenter) Reset()         { *m = Consenter{} }
func (m *Consenter) String() string { return proto.CompactTextString(m) }
func (*Consenter) ProtoMessage()    {}
func (*Consenter) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_5df0b1cd4230b47f, []int{1}
}
func (m *Consenter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Consenter.Unmarshal(m, b)
}
func (m *Consenter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Consenter.Marshal(b, m, deterministic)
}
func (dst *Consenter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Consenter.Merge(dst, src)
}
func (m *Consenter) XXX_Size() int {
	return xxx_messageInfo_Consenter.Size(m)
}
func (m *Consenter) XXX_DiscardUnknown() {
	xxx_messageInfo_Consenter.DiscardUnknown(m)
}

var xxx_messageInfo_Consenter proto.InternalMessageInfo

func (m *Consenter) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *Consenter) GetPort() uint32 {
	if m != nil {
		return m.Port
	}
	return 0
}

func (m *Consenter) GetClientTlsCert() []byte {
	if m != nil {
		return m.ClientTlsCert
	}
	return nil
}

func (m *Consenter) GetServerTlsCert() []byte {
	if m != nil {
		return m.ServerTlsCert
	}
	return nil
}

// FSMConfig enumerates the configuration options required for etcd/raft FSM.
type FSMConfig struct {
	// specified in miliseconds
	TickInterval         uint32   `protobuf:"varint,1,opt,name=TickInterval" json:"TickInterval,omitempty"`
	ElectionTick         uint32   `protobuf:"varint,2,opt,name=ElectionTick" json:"ElectionTick,omitempty"`
	HeartbeatTick        uint32   `protobuf:"varint,3,opt,name=HeartbeatTick" json:"HeartbeatTick,omitempty"`
	MaxInflightMsgs      uint32   `protobuf:"varint,4,opt,name=MaxInflightMsgs" json:"MaxInflightMsgs,omitempty"`
	MaxSizePerMsg        uint64   `protobuf:"varint,5,opt,name=MaxSizePerMsg" json:"MaxSizePerMsg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FSMConfig) Reset()         { *m = FSMConfig{} }
func (m *FSMConfig) String() string { return proto.CompactTextString(m) }
func (*FSMConfig) ProtoMessage()    {}
func (*FSMConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_configuration_5df0b1cd4230b47f, []int{2}
}
func (m *FSMConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FSMConfig.Unmarshal(m, b)
}
func (m *FSMConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FSMConfig.Marshal(b, m, deterministic)
}
func (dst *FSMConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FSMConfig.Merge(dst, src)
}
func (m *FSMConfig) XXX_Size() int {
	return xxx_messageInfo_FSMConfig.Size(m)
}
func (m *FSMConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_FSMConfig.DiscardUnknown(m)
}

var xxx_messageInfo_FSMConfig proto.InternalMessageInfo

func (m *FSMConfig) GetTickInterval() uint32 {
	if m != nil {
		return m.TickInterval
	}
	return 0
}

func (m *FSMConfig) GetElectionTick() uint32 {
	if m != nil {
		return m.ElectionTick
	}
	return 0
}

func (m *FSMConfig) GetHeartbeatTick() uint32 {
	if m != nil {
		return m.HeartbeatTick
	}
	return 0
}

func (m *FSMConfig) GetMaxInflightMsgs() uint32 {
	if m != nil {
		return m.MaxInflightMsgs
	}
	return 0
}

func (m *FSMConfig) GetMaxSizePerMsg() uint64 {
	if m != nil {
		return m.MaxSizePerMsg
	}
	return 0
}

func init() {
	proto.RegisterType((*Metadata)(nil), "etcdraft.Metadata")
	proto.RegisterType((*Consenter)(nil), "etcdraft.Consenter")
	proto.RegisterType((*FSMConfig)(nil), "etcdraft.FSMConfig")
}

func init() {
	proto.RegisterFile("orderer/etcdraft/configuration.proto", fileDescriptor_configuration_5df0b1cd4230b47f)
}

var fileDescriptor_configuration_5df0b1cd4230b47f = []byte{
	// 363 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x92, 0xc1, 0x6a, 0xdc, 0x30,
	0x10, 0x86, 0x51, 0x77, 0x5b, 0xb2, 0x6a, 0x4c, 0x40, 0xbd, 0xf8, 0x68, 0x96, 0x50, 0x7c, 0x92,
	0x21, 0xa1, 0x2f, 0xd0, 0xa5, 0xa1, 0x39, 0x18, 0x8a, 0x36, 0xa7, 0x5e, 0x16, 0x59, 0x1e, 0xcb,
	0xa2, 0xae, 0xb5, 0x8c, 0x26, 0x21, 0xed, 0xb5, 0x4f, 0xd7, 0xb7, 0x0a, 0x96, 0x76, 0x37, 0xf1,
	0xde, 0x86, 0x6f, 0xbe, 0x19, 0xff, 0x58, 0xc3, 0xaf, 0x3d, 0xb6, 0x80, 0x80, 0x15, 0x90, 0x69,
	0x51, 0x77, 0x54, 0x19, 0x3f, 0x76, 0xce, 0x3e, 0xa2, 0x26, 0xe7, 0x47, 0xb9, 0x47, 0x4f, 0x5e,
	0x5c, 0x1c, 0xbb, 0xeb, 0xc0, 0x2f, 0x6a, 0x20, 0xdd, 0x6a, 0xd2, 0xe2, 0x96, 0x73, 0xe3, 0xc7,
	0x00, 0x23, 0x01, 0x86, 0x9c, 0x15, 0x8b, 0xf2, 0xe3, 0xcd, 0x27, 0x79, 0x54, 0xe5, 0xe6, 0xd8,
	0x53, 0x6f, 0x34, 0x71, 0xc3, 0xf9, 0xdd, 0xb6, 0xde, 0xa5, 0xaf, 0xe4, 0xef, 0x0a, 0x36, 0x1f,
	0xba, 0xdb, 0xd6, 0x9b, 0xd8, 0x52, 0xab, 0x53, 0xb9, 0xfe, 0xc7, 0xf8, 0xea, 0xb4, 0x4d, 0x08,
	0xbe, 0xec, 0x7d, 0xa0, 0x9c, 0x15, 0xac, 0x5c, 0xa9, 0x58, 0x4f, 0x6c, 0xef, 0x91, 0xe2, 0xbe,
	0x4c, 0xc5, 0x5a, 0x7c, 0xe6, 0x57, 0x66, 0x70, 0x30, 0xd2, 0x8e, 0x86, 0xb0, 0x33, 0x80, 0x94,
	0x2f, 0x0a, 0x56, 0x5e, 0xaa, 0x2c, 0xe1, 0x87, 0x21, 0x6c, 0x20, 0x79, 0x01, 0xf0, 0x09, 0xf0,
	0xd5, 0x5b, 0x26, 0x2f, 0xe1, 0x83, 0xb7, 0xfe, 0xcf, 0xf8, 0x6b, 0x26, 0xb1, 0xe6, 0x97, 0x0f,
	0xce, 0xfc, 0xba, 0x9f, 0x22, 0x3d, 0xe9, 0x21, 0xa6, 0xc9, 0xd4, 0x8c, 0x4d, 0xce, 0xb7, 0x01,
	0xcc, 0xf4, 0x23, 0x27, 0x7e, 0x48, 0x37, 0x63, 0xe2, 0x9a, 0x67, 0xdf, 0x41, 0x23, 0x35, 0xa0,
	0x29, 0x4a, 0x8b, 0x28, 0xcd, 0xa1, 0x28, 0xf9, 0x55, 0xad, 0x9f, 0xef, 0xc7, 0x6e, 0x70, 0xb6,
	0xa7, 0x3a, 0xd8, 0x10, 0x33, 0x66, 0xea, 0x1c, 0x4f, 0xfb, 0x6a, 0xfd, 0xbc, 0x75, 0x7f, 0xe1,
	0x07, 0x60, 0x1d, 0x6c, 0xfe, 0xbe, 0x60, 0xe5, 0x52, 0xcd, 0xe1, 0x57, 0xcb, 0xa5, 0x47, 0x2b,
	0xfb, 0x3f, 0x7b, 0xc0, 0x01, 0x5a, 0x0b, 0x28, 0x3b, 0xdd, 0xa0, 0x33, 0xe9, 0xc1, 0x83, 0x3c,
	0x9c, 0xc5, 0xe9, 0x61, 0x7e, 0x7e, 0xb1, 0x8e, 0xfa, 0xc7, 0x46, 0x1a, 0xff, 0xbb, 0x7a, 0x33,
	0x56, 0xa5, 0xb1, 0x2a, 0x8d, 0x55, 0xe7, 0xd7, 0xd4, 0x7c, 0x88, 0x8d, 0xdb, 0x97, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x4b, 0xd5, 0x2d, 0x76, 0x68, 0x02, 0x00, 0x00,
}
