// Code generated by protoc-gen-go.
// source: spotify.proto
// DO NOT EDIT!

/*
Package spotify is a generated protocol buffer package.

It is generated from these files:
	spotify.proto

It has these top-level messages:
	AuthState
	Token
	AuthToken
*/
package spotify

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

type AuthState struct {
	Id          string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	UserId      string `protobuf:"bytes,2,opt,name=user_id,json=userId" json:"user_id,omitempty"`
	ResponseUrl string `protobuf:"bytes,3,opt,name=response_url,json=responseUrl" json:"response_url,omitempty"`
	ExpireTime  int64  `protobuf:"varint,4,opt,name=expire_time,json=expireTime" json:"expire_time,omitempty"`
}

func (m *AuthState) Reset()                    { *m = AuthState{} }
func (m *AuthState) String() string            { return proto.CompactTextString(m) }
func (*AuthState) ProtoMessage()               {}
func (*AuthState) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *AuthState) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *AuthState) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *AuthState) GetResponseUrl() string {
	if m != nil {
		return m.ResponseUrl
	}
	return ""
}

func (m *AuthState) GetExpireTime() int64 {
	if m != nil {
		return m.ExpireTime
	}
	return 0
}

type Token struct {
	AccessToken  string `protobuf:"bytes,1,opt,name=access_token,json=accessToken" json:"access_token,omitempty"`
	TokenType    string `protobuf:"bytes,2,opt,name=token_type,json=tokenType" json:"token_type,omitempty"`
	RefreshToken string `protobuf:"bytes,3,opt,name=refresh_token,json=refreshToken" json:"refresh_token,omitempty"`
	ExpiresAt    int64  `protobuf:"varint,4,opt,name=expires_at,json=expiresAt" json:"expires_at,omitempty"`
}

func (m *Token) Reset()                    { *m = Token{} }
func (m *Token) String() string            { return proto.CompactTextString(m) }
func (*Token) ProtoMessage()               {}
func (*Token) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Token) GetAccessToken() string {
	if m != nil {
		return m.AccessToken
	}
	return ""
}

func (m *Token) GetTokenType() string {
	if m != nil {
		return m.TokenType
	}
	return ""
}

func (m *Token) GetRefreshToken() string {
	if m != nil {
		return m.RefreshToken
	}
	return ""
}

func (m *Token) GetExpiresAt() int64 {
	if m != nil {
		return m.ExpiresAt
	}
	return 0
}

type AuthToken struct {
	Token *Token `protobuf:"bytes,1,opt,name=token" json:"token,omitempty"`
}

func (m *AuthToken) Reset()                    { *m = AuthToken{} }
func (m *AuthToken) String() string            { return proto.CompactTextString(m) }
func (*AuthToken) ProtoMessage()               {}
func (*AuthToken) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *AuthToken) GetToken() *Token {
	if m != nil {
		return m.Token
	}
	return nil
}

func init() {
	proto.RegisterType((*AuthState)(nil), "spotify.AuthState")
	proto.RegisterType((*Token)(nil), "spotify.Token")
	proto.RegisterType((*AuthToken)(nil), "spotify.AuthToken")
}

func init() { proto.RegisterFile("spotify.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 242 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x44, 0x90, 0x41, 0x4f, 0xc2, 0x40,
	0x10, 0x85, 0xd3, 0x22, 0x90, 0x4e, 0x81, 0xc3, 0x5e, 0xec, 0xc5, 0x88, 0xd5, 0x03, 0x27, 0x12,
	0xf5, 0x17, 0x70, 0xf4, 0x5a, 0xeb, 0x79, 0x53, 0xe9, 0x10, 0x36, 0x02, 0xbb, 0xd9, 0x99, 0x26,
	0xf4, 0x47, 0xf8, 0x9f, 0x4d, 0x77, 0xb6, 0x72, 0x7c, 0xdf, 0x9b, 0xbc, 0x79, 0x79, 0xb0, 0x24,
	0x67, 0xd9, 0x1c, 0xfa, 0xad, 0xf3, 0x96, 0xad, 0x9a, 0x47, 0x59, 0x5e, 0x21, 0xdb, 0x75, 0x7c,
	0xfc, 0xe4, 0x86, 0x51, 0xad, 0x20, 0x35, 0x6d, 0x91, 0xac, 0x93, 0x4d, 0x56, 0xa5, 0xa6, 0x55,
	0xf7, 0x30, 0xef, 0x08, 0xbd, 0x36, 0x6d, 0x91, 0x06, 0x38, 0x1b, 0xe4, 0x47, 0xab, 0x9e, 0x60,
	0xe1, 0x91, 0x9c, 0xbd, 0x10, 0xea, 0xce, 0x9f, 0x8a, 0x49, 0x70, 0xf3, 0x91, 0x7d, 0xf9, 0x93,
	0x7a, 0x84, 0x1c, 0xaf, 0xce, 0x78, 0xd4, 0x6c, 0xce, 0x58, 0xdc, 0xad, 0x93, 0xcd, 0xa4, 0x02,
	0x41, 0xb5, 0x39, 0x63, 0xf9, 0x9b, 0xc0, 0xb4, 0xb6, 0x3f, 0x78, 0x19, 0xd2, 0x9a, 0xfd, 0x1e,
	0x89, 0x34, 0x0f, 0x3a, 0x16, 0xc8, 0x85, 0xc9, 0xc9, 0x03, 0x40, 0xf0, 0x34, 0xf7, 0x0e, 0x63,
	0x99, 0x2c, 0x90, 0xba, 0x77, 0xa8, 0x9e, 0x61, 0xe9, 0xf1, 0xe0, 0x91, 0x8e, 0x31, 0x42, 0x0a,
	0x2d, 0x22, 0xfc, 0xcf, 0x90, 0xf7, 0xa4, 0x1b, 0x8e, 0x85, 0xb2, 0x48, 0x76, 0x5c, 0xbe, 0xca,
	0x12, 0x72, 0xfb, 0x02, 0xd3, 0x5b, 0x97, 0xfc, 0x6d, 0xb5, 0x1d, 0xe7, 0x0b, 0x76, 0x25, 0xe6,
	0xf7, 0x2c, 0x8c, 0xf9, 0xfe, 0x17, 0x00, 0x00, 0xff, 0xff, 0xba, 0xa0, 0x70, 0x1b, 0x5d, 0x01,
	0x00, 0x00,
}