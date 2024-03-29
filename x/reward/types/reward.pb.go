// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: xpla/reward/v1beta1/reward.proto

package types

import (
	fmt "fmt"
	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Params defines the set of params for the reward module.
type Params struct {
	FeePoolRate             github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=fee_pool_rate,json=feePoolRate,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"fee_pool_rate" yaml:"fee_pool_rate"`
	CommunityPoolRate       github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,2,opt,name=community_pool_rate,json=communityPoolRate,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"community_pool_rate" yaml:"community_pool_rate"`
	ReserveRate             github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,3,opt,name=reserve_rate,json=reserveRate,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"reserve_rate" yaml:"reserve_rate"`
	ReserveAccount          string                                 `protobuf:"bytes,4,opt,name=reserve_account,json=reserveAccount,proto3" json:"reserve_account,omitempty"`
	RewardDistributeAccount string                                 `protobuf:"bytes,5,opt,name=reward_distribute_account,json=rewardDistributeAccount,proto3" json:"reward_distribute_account,omitempty"`
}

func (m *Params) Reset()      { *m = Params{} }
func (*Params) ProtoMessage() {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_cce4bfd3ebfaf11e, []int{0}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetReserveAccount() string {
	if m != nil {
		return m.ReserveAccount
	}
	return ""
}

func (m *Params) GetRewardDistributeAccount() string {
	if m != nil {
		return m.RewardDistributeAccount
	}
	return ""
}

func init() {
	proto.RegisterType((*Params)(nil), "xpla.reward.v1beta1.Params")
}

func init() { proto.RegisterFile("xpla/reward/v1beta1/reward.proto", fileDescriptor_cce4bfd3ebfaf11e) }

var fileDescriptor_cce4bfd3ebfaf11e = []byte{
	// 352 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xa8, 0x28, 0xc8, 0x49,
	0xd4, 0x2f, 0x4a, 0x2d, 0x4f, 0x2c, 0x4a, 0xd1, 0x2f, 0x33, 0x4c, 0x4a, 0x2d, 0x49, 0x34, 0x84,
	0x72, 0xf5, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0x84, 0x41, 0x2a, 0xf4, 0xa0, 0x42, 0x50, 0x15,
	0x52, 0x22, 0xe9, 0xf9, 0xe9, 0xf9, 0x60, 0x79, 0x7d, 0x10, 0x0b, 0xa2, 0x54, 0xe9, 0x38, 0x33,
	0x17, 0x5b, 0x40, 0x62, 0x51, 0x62, 0x6e, 0xb1, 0x50, 0x16, 0x17, 0x6f, 0x5a, 0x6a, 0x6a, 0x7c,
	0x41, 0x7e, 0x7e, 0x4e, 0x7c, 0x51, 0x62, 0x49, 0xaa, 0x04, 0xa3, 0x02, 0xa3, 0x06, 0xa7, 0x93,
	0xdb, 0x89, 0x7b, 0xf2, 0x0c, 0xb7, 0xee, 0xc9, 0xab, 0xa5, 0x67, 0x96, 0x64, 0x94, 0x26, 0xe9,
	0x25, 0xe7, 0xe7, 0xea, 0x27, 0xe7, 0x17, 0xe7, 0xe6, 0x17, 0x43, 0x29, 0xdd, 0xe2, 0x94, 0x6c,
	0xfd, 0x92, 0xca, 0x82, 0xd4, 0x62, 0x3d, 0x97, 0xd4, 0xe4, 0x4f, 0xf7, 0xe4, 0x45, 0x2a, 0x13,
	0x73, 0x73, 0xac, 0x94, 0x50, 0x0c, 0x53, 0x0a, 0xe2, 0x4e, 0x4b, 0x4d, 0x0d, 0xc8, 0xcf, 0xcf,
	0x09, 0x4a, 0x2c, 0x49, 0x15, 0xaa, 0xe1, 0x12, 0x4e, 0xce, 0xcf, 0xcd, 0x2d, 0xcd, 0xcb, 0x2c,
	0xa9, 0x44, 0xb2, 0x91, 0x09, 0x6c, 0xa3, 0x0f, 0xc9, 0x36, 0x4a, 0x41, 0x6c, 0xc4, 0x62, 0xa4,
	0x52, 0x90, 0x20, 0x5c, 0x14, 0x6e, 0x7b, 0x06, 0x17, 0x4f, 0x51, 0x6a, 0x71, 0x6a, 0x51, 0x59,
	0x2a, 0xc4, 0x5a, 0x66, 0xb0, 0xb5, 0xae, 0x24, 0x5b, 0x2b, 0x0c, 0xb1, 0x16, 0xd9, 0x2c, 0xa5,
	0x20, 0x6e, 0x28, 0x17, 0x6c, 0x93, 0x3a, 0x17, 0x3f, 0x4c, 0x36, 0x31, 0x39, 0x39, 0xbf, 0x34,
	0xaf, 0x44, 0x82, 0x05, 0x64, 0x59, 0x10, 0x1f, 0x54, 0xd8, 0x11, 0x22, 0x2a, 0x64, 0xc5, 0x25,
	0x09, 0x89, 0xaf, 0xf8, 0x94, 0xcc, 0xe2, 0x92, 0xa2, 0xcc, 0xa4, 0xd2, 0x12, 0x84, 0x16, 0x56,
	0xb0, 0x16, 0x71, 0x88, 0x02, 0x17, 0xb8, 0x3c, 0x54, 0xaf, 0x15, 0xcb, 0x8c, 0x05, 0xf2, 0x0c,
	0x4e, 0x0e, 0x27, 0x1e, 0xc9, 0x31, 0x5e, 0x78, 0x24, 0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x84,
	0xc7, 0x72, 0x0c, 0x17, 0x1e, 0xcb, 0x31, 0xdc, 0x78, 0x2c, 0xc7, 0x10, 0x85, 0xec, 0x21, 0x50,
	0xca, 0x48, 0x49, 0x2d, 0x03, 0xd3, 0xfa, 0x15, 0xb0, 0x54, 0x04, 0xf6, 0x54, 0x12, 0x1b, 0x38,
	0x49, 0x18, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xac, 0x0c, 0x1b, 0x7b, 0x61, 0x02, 0x00, 0x00,
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.RewardDistributeAccount) > 0 {
		i -= len(m.RewardDistributeAccount)
		copy(dAtA[i:], m.RewardDistributeAccount)
		i = encodeVarintReward(dAtA, i, uint64(len(m.RewardDistributeAccount)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.ReserveAccount) > 0 {
		i -= len(m.ReserveAccount)
		copy(dAtA[i:], m.ReserveAccount)
		i = encodeVarintReward(dAtA, i, uint64(len(m.ReserveAccount)))
		i--
		dAtA[i] = 0x22
	}
	{
		size := m.ReserveRate.Size()
		i -= size
		if _, err := m.ReserveRate.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReward(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	{
		size := m.CommunityPoolRate.Size()
		i -= size
		if _, err := m.CommunityPoolRate.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReward(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	{
		size := m.FeePoolRate.Size()
		i -= size
		if _, err := m.FeePoolRate.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintReward(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintReward(dAtA []byte, offset int, v uint64) int {
	offset -= sovReward(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.FeePoolRate.Size()
	n += 1 + l + sovReward(uint64(l))
	l = m.CommunityPoolRate.Size()
	n += 1 + l + sovReward(uint64(l))
	l = m.ReserveRate.Size()
	n += 1 + l + sovReward(uint64(l))
	l = len(m.ReserveAccount)
	if l > 0 {
		n += 1 + l + sovReward(uint64(l))
	}
	l = len(m.RewardDistributeAccount)
	if l > 0 {
		n += 1 + l + sovReward(uint64(l))
	}
	return n
}

func sovReward(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozReward(x uint64) (n int) {
	return sovReward(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReward
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FeePoolRate", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.FeePoolRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CommunityPoolRate", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.CommunityPoolRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReserveRate", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ReserveRate.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReserveAccount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ReserveAccount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RewardDistributeAccount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReward
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthReward
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthReward
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RewardDistributeAccount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReward(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReward
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipReward(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowReward
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowReward
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowReward
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthReward
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupReward
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthReward
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthReward        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowReward          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupReward = fmt.Errorf("proto: unexpected end of group")
)
