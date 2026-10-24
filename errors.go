package lzfse

import "errors"

var (
	// ErrShortBuffer is returned when the destination buffer cannot hold the
	// complete result. Compress with a buffer of at least
	// cap(src)+cap(src)/2+1024 bytes (what EncodeBuffer allocates), and
	// decompress into a buffer at least as large as the original input.
	ErrShortBuffer = errors.New("lzfse: destination buffer is too short")

	// ErrDecompress is returned when the native decoder rejects the input.
	ErrDecompress = errors.New("lzfse: decompression failed")
)
