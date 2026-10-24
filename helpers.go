package lzfse

import (
	"encoding/binary"
	"runtime"
	"unsafe"
)

// sliceData returns the address of the first byte of b, or 0 when b is empty.
func sliceData(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(b)))
}

// keepAlive pins a Go value across a call into native code that received the
// value's address as a uintptr.
func keepAlive(v any) {
	runtime.KeepAlive(v)
}

// encodeDestSize returns a destination buffer size that lzfse_encode_buffer
// can always fill successfully for an input of n bytes. The worst case is
// bounded by the largest magic the container can emit for a block plus block
// headers; n + n/2 + 1024 has ample headroom.
func encodeDestSize(n int) int {
	if n < 1024 {
		n = 1024
	}
	return n + n/2 + 1024
}

// decodeDestSize returns the initial destination buffer size DecodeBuffer
// tries for compressed input of n bytes; it grows geometrically when needed.
func decodeDestSize(n int) int {
	if n == 0 {
		return 1024
	}
	return n * 4
}

// peekDecodedSize returns the total decoded size of an LZFSE stream when it
// is readable from the first block header, or 0. Both the "bvx-" raw block
// and the "bvxn" LZVN block header store the block's decoded byte count at
// offset 4 (little endian uint32); a stream made of a single such block, as
// the encoder emits for inputs that fit one block, is fully sized this way.
func peekDecodedSize(src []byte) int {
	if len(src) < 8 {
		return 0
	}
	magic := binary.LittleEndian.Uint32(src)
	// 0x2d787662 = "bvx-" (raw block), 0x6e787662 = "bvxn" (LZVN block).
	if magic != 0x2d787662 && magic != 0x6e787662 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(src[4:8]))
}
