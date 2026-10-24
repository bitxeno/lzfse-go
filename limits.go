package lzfse

// maxDecodeSize bounds how far DecodeBuffer grows its destination buffer
// while looking for the decompressed size, so malformed input cannot make it
// allocate unboundedly. LZFSE rarely expands more than ~1000x.
func maxDecodeSize(compressedSize int) int {
	const floor = 256 << 20
	limit := compressedSize << 10
	if limit < floor {
		return floor
	}
	return limit
}
