// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: waku_metadata.proto

// rfc: https://rfc.vac.dev/spec/66/

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type WakuMetadataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId *uint32  `protobuf:"varint,1,opt,name=cluster_id,json=clusterId,proto3,oneof" json:"cluster_id,omitempty"`
	Shards    []uint32 `protobuf:"varint,3,rep,packed,name=shards,proto3" json:"shards,omitempty"`
	// Starting from nwaku v0.26, if field 3 contains no data, it will attempt to
	// decode this field first assuming it's a packed field, and if that fails,
	// attempt to decode as an unpacked field
	//
	// Deprecated: Marked as deprecated in waku_metadata.proto.
	ShardsDeprecated []uint32 `protobuf:"varint,2,rep,name=shards_deprecated,json=shardsDeprecated,proto3" json:"shards_deprecated,omitempty"`
}

func (x *WakuMetadataRequest) Reset() {
	*x = WakuMetadataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waku_metadata_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WakuMetadataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WakuMetadataRequest) ProtoMessage() {}

func (x *WakuMetadataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waku_metadata_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WakuMetadataRequest.ProtoReflect.Descriptor instead.
func (*WakuMetadataRequest) Descriptor() ([]byte, []int) {
	return file_waku_metadata_proto_rawDescGZIP(), []int{0}
}

func (x *WakuMetadataRequest) GetClusterId() uint32 {
	if x != nil && x.ClusterId != nil {
		return *x.ClusterId
	}
	return 0
}

func (x *WakuMetadataRequest) GetShards() []uint32 {
	if x != nil {
		return x.Shards
	}
	return nil
}

// Deprecated: Marked as deprecated in waku_metadata.proto.
func (x *WakuMetadataRequest) GetShardsDeprecated() []uint32 {
	if x != nil {
		return x.ShardsDeprecated
	}
	return nil
}

type WakuMetadataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId *uint32  `protobuf:"varint,1,opt,name=cluster_id,json=clusterId,proto3,oneof" json:"cluster_id,omitempty"`
	Shards    []uint32 `protobuf:"varint,3,rep,packed,name=shards,proto3" json:"shards,omitempty"`
	// Starting from nwaku v0.26, if field 3 contains no data, it will attempt to
	// decode this field first assuming it's a packed field, and if that fails,
	// attempt to decode as an unpacked field
	//
	// Deprecated: Marked as deprecated in waku_metadata.proto.
	ShardsDeprecated []uint32 `protobuf:"varint,2,rep,name=shards_deprecated,json=shardsDeprecated,proto3" json:"shards_deprecated,omitempty"`
}

func (x *WakuMetadataResponse) Reset() {
	*x = WakuMetadataResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waku_metadata_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WakuMetadataResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WakuMetadataResponse) ProtoMessage() {}

func (x *WakuMetadataResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waku_metadata_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WakuMetadataResponse.ProtoReflect.Descriptor instead.
func (*WakuMetadataResponse) Descriptor() ([]byte, []int) {
	return file_waku_metadata_proto_rawDescGZIP(), []int{1}
}

func (x *WakuMetadataResponse) GetClusterId() uint32 {
	if x != nil && x.ClusterId != nil {
		return *x.ClusterId
	}
	return 0
}

func (x *WakuMetadataResponse) GetShards() []uint32 {
	if x != nil {
		return x.Shards
	}
	return nil
}

// Deprecated: Marked as deprecated in waku_metadata.proto.
func (x *WakuMetadataResponse) GetShardsDeprecated() []uint32 {
	if x != nil {
		return x.ShardsDeprecated
	}
	return nil
}

var File_waku_metadata_proto protoreflect.FileDescriptor

var file_waku_metadata_proto_rawDesc = []byte{
	0x0a, 0x13, 0x77, 0x61, 0x6b, 0x75, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10, 0x77, 0x61, 0x6b, 0x75, 0x2e, 0x6d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x2e, 0x76, 0x31, 0x22, 0x93, 0x01, 0x0a, 0x13, 0x57, 0x61, 0x6b, 0x75,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x22, 0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0d, 0x48, 0x00, 0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64,
	0x88, 0x01, 0x01, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x68, 0x61, 0x72, 0x64, 0x73, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x0d, 0x52, 0x06, 0x73, 0x68, 0x61, 0x72, 0x64, 0x73, 0x12, 0x31, 0x0a, 0x11, 0x73,
	0x68, 0x61, 0x72, 0x64, 0x73, 0x5f, 0x64, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64,
	0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x42, 0x04, 0x10, 0x00, 0x18, 0x01, 0x52, 0x10, 0x73, 0x68,
	0x61, 0x72, 0x64, 0x73, 0x44, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x42, 0x0d,
	0x0a, 0x0b, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x22, 0x94, 0x01,
	0x0a, 0x14, 0x57, 0x61, 0x6b, 0x75, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x22, 0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x48, 0x00, 0x52, 0x09, 0x63, 0x6c,
	0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64, 0x88, 0x01, 0x01, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x68,
	0x61, 0x72, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x06, 0x73, 0x68, 0x61, 0x72,
	0x64, 0x73, 0x12, 0x31, 0x0a, 0x11, 0x73, 0x68, 0x61, 0x72, 0x64, 0x73, 0x5f, 0x64, 0x65, 0x70,
	0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x42, 0x04, 0x10,
	0x00, 0x18, 0x01, 0x52, 0x10, 0x73, 0x68, 0x61, 0x72, 0x64, 0x73, 0x44, 0x65, 0x70, 0x72, 0x65,
	0x63, 0x61, 0x74, 0x65, 0x64, 0x42, 0x0d, 0x0a, 0x0b, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x5f, 0x69, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waku_metadata_proto_rawDescOnce sync.Once
	file_waku_metadata_proto_rawDescData = file_waku_metadata_proto_rawDesc
)

func file_waku_metadata_proto_rawDescGZIP() []byte {
	file_waku_metadata_proto_rawDescOnce.Do(func() {
		file_waku_metadata_proto_rawDescData = protoimpl.X.CompressGZIP(file_waku_metadata_proto_rawDescData)
	})
	return file_waku_metadata_proto_rawDescData
}

var file_waku_metadata_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_waku_metadata_proto_goTypes = []interface{}{
	(*WakuMetadataRequest)(nil),  // 0: waku.metadata.v1.WakuMetadataRequest
	(*WakuMetadataResponse)(nil), // 1: waku.metadata.v1.WakuMetadataResponse
}
var file_waku_metadata_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_waku_metadata_proto_init() }
func file_waku_metadata_proto_init() {
	if File_waku_metadata_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_waku_metadata_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WakuMetadataRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waku_metadata_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WakuMetadataResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_waku_metadata_proto_msgTypes[0].OneofWrappers = []interface{}{}
	file_waku_metadata_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_waku_metadata_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_waku_metadata_proto_goTypes,
		DependencyIndexes: file_waku_metadata_proto_depIdxs,
		MessageInfos:      file_waku_metadata_proto_msgTypes,
	}.Build()
	File_waku_metadata_proto = out.File
	file_waku_metadata_proto_rawDesc = nil
	file_waku_metadata_proto_goTypes = nil
	file_waku_metadata_proto_depIdxs = nil
}
