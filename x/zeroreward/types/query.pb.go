// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: xpla/zeroreward/v1beta1/query.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	grpc1 "github.com/gogo/protobuf/grpc"
	proto "github.com/gogo/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type QueryZeroRewardValidatorsRequest struct {
}

func (m *QueryZeroRewardValidatorsRequest) Reset()         { *m = QueryZeroRewardValidatorsRequest{} }
func (m *QueryZeroRewardValidatorsRequest) String() string { return proto.CompactTextString(m) }
func (*QueryZeroRewardValidatorsRequest) ProtoMessage()    {}
func (*QueryZeroRewardValidatorsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_76a01e16a068365f, []int{0}
}
func (m *QueryZeroRewardValidatorsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryZeroRewardValidatorsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryZeroRewardValidatorsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryZeroRewardValidatorsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryZeroRewardValidatorsRequest.Merge(m, src)
}
func (m *QueryZeroRewardValidatorsRequest) XXX_Size() int {
	return m.Size()
}
func (m *QueryZeroRewardValidatorsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryZeroRewardValidatorsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_QueryZeroRewardValidatorsRequest proto.InternalMessageInfo

type QueryZeroRewardValidatorsResponse struct {
	ZeroRewardValidators []string `protobuf:"bytes,1,rep,name=zero_reward_validators,json=zeroRewardValidators,proto3" json:"zero_reward_validators,omitempty"`
}

func (m *QueryZeroRewardValidatorsResponse) Reset()         { *m = QueryZeroRewardValidatorsResponse{} }
func (m *QueryZeroRewardValidatorsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryZeroRewardValidatorsResponse) ProtoMessage()    {}
func (*QueryZeroRewardValidatorsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_76a01e16a068365f, []int{1}
}
func (m *QueryZeroRewardValidatorsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *QueryZeroRewardValidatorsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryZeroRewardValidatorsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *QueryZeroRewardValidatorsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryZeroRewardValidatorsResponse.Merge(m, src)
}
func (m *QueryZeroRewardValidatorsResponse) XXX_Size() int {
	return m.Size()
}
func (m *QueryZeroRewardValidatorsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryZeroRewardValidatorsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryZeroRewardValidatorsResponse proto.InternalMessageInfo

func (m *QueryZeroRewardValidatorsResponse) GetZeroRewardValidators() []string {
	if m != nil {
		return m.ZeroRewardValidators
	}
	return nil
}

func init() {
	proto.RegisterType((*QueryZeroRewardValidatorsRequest)(nil), "xpla.zeroreward.v1beta1.QueryZeroRewardValidatorsRequest")
	proto.RegisterType((*QueryZeroRewardValidatorsResponse)(nil), "xpla.zeroreward.v1beta1.QueryZeroRewardValidatorsResponse")
}

func init() {
	proto.RegisterFile("xpla/zeroreward/v1beta1/query.proto", fileDescriptor_76a01e16a068365f)
}

var fileDescriptor_76a01e16a068365f = []byte{
	// 285 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0xae, 0x28, 0xc8, 0x49,
	0xd4, 0xaf, 0x4a, 0x2d, 0xca, 0x2f, 0x4a, 0x2d, 0x4f, 0x2c, 0x4a, 0xd1, 0x2f, 0x33, 0x4c, 0x4a,
	0x2d, 0x49, 0x34, 0xd4, 0x2f, 0x2c, 0x4d, 0x2d, 0xaa, 0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17,
	0x12, 0x07, 0x29, 0xd2, 0x43, 0x28, 0xd2, 0x83, 0x2a, 0x92, 0x12, 0x49, 0xcf, 0x4f, 0xcf, 0x07,
	0xab, 0xd1, 0x07, 0xb1, 0x20, 0xca, 0xa5, 0x64, 0xd2, 0xf3, 0xf3, 0xd3, 0x73, 0x52, 0xf5, 0x13,
	0x0b, 0x32, 0xf5, 0x13, 0xf3, 0xf2, 0xf2, 0x4b, 0x12, 0x4b, 0x32, 0xf3, 0xf3, 0x8a, 0x21, 0xb2,
	0x4a, 0x4a, 0x5c, 0x0a, 0x81, 0x20, 0xb3, 0xa3, 0x52, 0x8b, 0xf2, 0x83, 0xc0, 0xc6, 0x85, 0x25,
	0xe6, 0x64, 0xa6, 0x24, 0x96, 0xe4, 0x17, 0x15, 0x07, 0xa5, 0x16, 0x96, 0xa6, 0x16, 0x97, 0x28,
	0x45, 0x72, 0x29, 0xe2, 0x51, 0x53, 0x5c, 0x90, 0x9f, 0x57, 0x9c, 0x2a, 0x64, 0xc2, 0x25, 0x06,
	0x72, 0x52, 0x3c, 0xc4, 0x4d, 0xf1, 0x65, 0x70, 0x15, 0x12, 0x8c, 0x0a, 0xcc, 0x1a, 0x9c, 0x41,
	0x22, 0x55, 0x58, 0x74, 0x1b, 0x1d, 0x61, 0xe4, 0x62, 0x05, 0x9b, 0x2d, 0xb4, 0x8b, 0x91, 0x4b,
	0x04, 0x9b, 0x05, 0x42, 0x96, 0x7a, 0x38, 0xfc, 0xab, 0x47, 0xc8, 0xe1, 0x52, 0x56, 0xe4, 0x68,
	0x85, 0xf8, 0x47, 0x49, 0xbb, 0xe9, 0xf2, 0x93, 0xc9, 0x4c, 0xaa, 0x42, 0xca, 0xfa, 0xb8, 0xe2,
	0x04, 0xe1, 0x45, 0x27, 0x97, 0x13, 0x8f, 0xe4, 0x18, 0x2f, 0x3c, 0x92, 0x63, 0x7c, 0xf0, 0x48,
	0x8e, 0x71, 0xc2, 0x63, 0x39, 0x86, 0x0b, 0x8f, 0xe5, 0x18, 0x6e, 0x3c, 0x96, 0x63, 0x88, 0xd2,
	0x4a, 0xcf, 0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0x05, 0x1b, 0x94, 0x92, 0x5a, 0x06,
	0x31, 0xb0, 0x02, 0xd9, 0xc8, 0x92, 0xca, 0x82, 0xd4, 0xe2, 0x24, 0x36, 0x70, 0x94, 0x18, 0x03,
	0x02, 0x00, 0x00, 0xff, 0xff, 0x4d, 0xc7, 0x9a, 0xc4, 0x06, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type QueryClient interface {
	ZeroRewardValidators(ctx context.Context, in *QueryZeroRewardValidatorsRequest, opts ...grpc.CallOption) (*QueryZeroRewardValidatorsResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) ZeroRewardValidators(ctx context.Context, in *QueryZeroRewardValidatorsRequest, opts ...grpc.CallOption) (*QueryZeroRewardValidatorsResponse, error) {
	out := new(QueryZeroRewardValidatorsResponse)
	err := c.cc.Invoke(ctx, "/xpla.zeroreward.v1beta1.Query/ZeroRewardValidators", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
type QueryServer interface {
	ZeroRewardValidators(context.Context, *QueryZeroRewardValidatorsRequest) (*QueryZeroRewardValidatorsResponse, error)
}

// UnimplementedQueryServer can be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (*UnimplementedQueryServer) ZeroRewardValidators(ctx context.Context, req *QueryZeroRewardValidatorsRequest) (*QueryZeroRewardValidatorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ZeroRewardValidators not implemented")
}

func RegisterQueryServer(s grpc1.Server, srv QueryServer) {
	s.RegisterService(&_Query_serviceDesc, srv)
}

func _Query_ZeroRewardValidators_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryZeroRewardValidatorsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ZeroRewardValidators(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/xpla.zeroreward.v1beta1.Query/ZeroRewardValidators",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ZeroRewardValidators(ctx, req.(*QueryZeroRewardValidatorsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Query_serviceDesc = grpc.ServiceDesc{
	ServiceName: "xpla.zeroreward.v1beta1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ZeroRewardValidators",
			Handler:    _Query_ZeroRewardValidators_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "xpla/zeroreward/v1beta1/query.proto",
}

func (m *QueryZeroRewardValidatorsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryZeroRewardValidatorsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryZeroRewardValidatorsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *QueryZeroRewardValidatorsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryZeroRewardValidatorsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryZeroRewardValidatorsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ZeroRewardValidators) > 0 {
		for iNdEx := len(m.ZeroRewardValidators) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.ZeroRewardValidators[iNdEx])
			copy(dAtA[i:], m.ZeroRewardValidators[iNdEx])
			i = encodeVarintQuery(dAtA, i, uint64(len(m.ZeroRewardValidators[iNdEx])))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintQuery(dAtA []byte, offset int, v uint64) int {
	offset -= sovQuery(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *QueryZeroRewardValidatorsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *QueryZeroRewardValidatorsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.ZeroRewardValidators) > 0 {
		for _, s := range m.ZeroRewardValidators {
			l = len(s)
			n += 1 + l + sovQuery(uint64(l))
		}
	}
	return n
}

func sovQuery(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozQuery(x uint64) (n int) {
	return sovQuery(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *QueryZeroRewardValidatorsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: QueryZeroRewardValidatorsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryZeroRewardValidatorsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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
func (m *QueryZeroRewardValidatorsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: QueryZeroRewardValidatorsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryZeroRewardValidatorsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ZeroRewardValidators", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
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
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ZeroRewardValidators = append(m.ZeroRewardValidators, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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
func skipQuery(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
				return 0, ErrInvalidLengthQuery
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQuery
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQuery
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQuery        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQuery          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQuery = fmt.Errorf("proto: unexpected end of group")
)