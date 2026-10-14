package kerlong

import (
	"bytes"
	"encoding/binary"
)

// BytesCombine 连接字符串
func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, nil)
}

// WriteUint8 写入1字节uint8
func WriteUint8(b *[]byte, i uint8, offset *int) {
	(*b)[*offset] = i
	(*offset)++
}

// WriteUint16 写入2字节uint16
func WriteUint16(b *[]byte, i uint16, offset *int) {
	(*b)[*offset] = uint8(i)
	(*b)[*offset+1] = uint8(i >> 8)
	(*offset) += 2
}

// WriteUint32 写入4字节uint32
func WriteUint32(b *[]byte, i uint32, offset *int) {
	(*b)[*offset] = uint8(i)
	(*b)[*offset+1] = uint8(i >> 8)
	(*b)[*offset+2] = uint8(i >> 16)
	(*b)[*offset+3] = uint8(i >> 24)
	(*offset) += 4
}

// WriteUint64 写入8字节uint64
func WriteUint64(b *[]byte, i uint64, offset *int) {
	(*b)[*offset] = uint8(i)
	(*b)[*offset+1] = uint8(i >> 8)
	(*b)[*offset+2] = uint8(i >> 16)
	(*b)[*offset+3] = uint8(i >> 24)
	(*b)[*offset+4] = uint8(i >> 32)
	(*b)[*offset+5] = uint8(i >> 40)
	(*b)[*offset+6] = uint8(i >> 48)
	(*b)[*offset+7] = uint8(i >> 56)
	(*offset) += 8
}

// WriteUint32BE 写入4字节uint32，大端模式
func WriteUint32BE(b *[]byte, i uint32, offset *int) {
	(*b)[*offset] = uint8(i >> 24)
	(*b)[*offset+1] = uint8(i >> 16)
	(*b)[*offset+2] = uint8(i >> 8)
	(*b)[*offset+3] = uint8(i)
	(*offset) += 4
}

// WriteString 写入字符串，包括长度
func WriteString(dest *[]byte, src []byte, offset *int) int {
	l := len(src)
	WriteUint8(dest, uint8(l), offset)
	if l == 0 {
		return 1
	}
	for i := 0; i < l; i++ {
		(*dest)[*offset] = src[i]
		(*offset)++
	}
	return l + 1
}

// WriteLongString 写入字符串，包括2字节长度
func WriteLongString(dest *[]byte, src []byte, offset *int) int {
	l := len(src)
	WriteUint16(dest, uint16(l), offset)
	if l == 0 {
		return 2
	}
	for i := 0; i < l; i++ {
		(*dest)[*offset] = src[i]
		(*offset)++
	}
	return l + 2
}

// ReadUint8 读取1字节到uint8
func ReadUint8(b []byte, offset *int) uint8 {
	buf := bytes.NewBuffer(b[(*offset):])
	var i uint8
	binary.Read(buf, binary.LittleEndian, &i)
	(*offset)++
	return i
}

// ReadUint16 读取2字节到uint16
func ReadUint16(b []byte, offset *int) uint16 {
	buf := bytes.NewBuffer(b[(*offset):])
	var i uint16
	binary.Read(buf, binary.LittleEndian, &i)
	(*offset) += 2
	return i
}

// ReadUint32 读取4字节到uint32
func ReadUint32(b []byte, offset *int) uint32 {
	buf := bytes.NewBuffer(b[(*offset):])
	var i uint32
	binary.Read(buf, binary.LittleEndian, &i)
	(*offset) += 4
	return i
}

// ReadUint64 读取8字节到uint64
func ReadUint64(b []byte, offset *int) uint64 {
	buf := bytes.NewBuffer(b[(*offset):])
	var i uint64
	binary.Read(buf, binary.LittleEndian, &i)
	(*offset) += 8
	return i
}

// ReadUint32BE 大端读取4字节到uint32
func ReadUint32BE(b []byte, offset *int) uint32 {
	buf := bytes.NewBuffer(b[(*offset):])
	var i uint32
	binary.Read(buf, binary.BigEndian, &i)
	(*offset) += 4
	return i
}

// ReadString 大端不会读取长度，需要单独先读取长度
func ReadString(b []byte, offset *int, len int) []byte {
	(*offset) += len
	return b[(*offset)-len : (*offset)]
}

// CompareBytes 比较两个数组是否内容相等,相等返回true
func CompareBytes(a []byte, b []byte) bool { return bytes.Equal(a, b) }

// BuildLongString 构建长字符串消息，包含2字节的字符串大小
func BuildLongString(src []byte) []byte {
	buf := make([]byte, 2)
	offset := 0
	WriteUint16(&buf, uint16(len(src)), &offset)
	return BytesCombine(buf, src)
}

// BuildString 构建长字符串消息，包含2字节的字符串大小
func BuildString(src []byte) []byte {
	buf := make([]byte, 1)
	offset := 0
	WriteUint8(&buf, uint8(len(src)), &offset)
	return BytesCombine(buf, src)
}
