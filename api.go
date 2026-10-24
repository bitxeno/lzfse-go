package lzfse

// Package-level convenience functions that use the default Library. They
// load the native shared library on first call; see the Library methods for
// details.

// EncodeScratchSize returns the required scratch buffer size to compress
// using LZFSE.
func EncodeScratchSize() (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.EncodeScratchSize(), nil
}

// DecodeScratchSize returns the required scratch buffer size to decompress
// using LZFSE.
func DecodeScratchSize() (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.DecodeScratchSize(), nil
}

// EncodeBuffer compresses src using LZFSE into a newly allocated buffer,
// returning the compressed bytes.
func EncodeBuffer(src []byte) ([]byte, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return nil, err
	}
	return lib.EncodeBuffer(src)
}

// EncodeBufferInto compresses src into dst and returns the number of bytes
// written. It returns ErrShortBuffer if dst is too small.
func EncodeBufferInto(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.EncodeBufferInto(src, dst)
}

// DecodeBuffer decompresses LZFSE-encoded src into a newly allocated buffer,
// returning the decompressed bytes.
func DecodeBuffer(src []byte) ([]byte, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return nil, err
	}
	return lib.DecodeBuffer(src)
}

// DecodeBufferInto decompresses LZFSE-encoded src into dst, returning the
// number of bytes written. It returns ErrShortBuffer if dst is too small.
func DecodeBufferInto(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.DecodeBufferInto(src, dst)
}

// EncodeLZVNBuffer compresses src using the LZVN block format into dst,
// returning the number of bytes written, or 0 if dst is too small.
func EncodeLZVNBuffer(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.EncodeLZVNBuffer(src, dst), nil
}

// DecodeLZVNBuffer decompresses LZVN-encoded src into dst, returning the
// number of bytes written, or 0 on error or if dst is too small.
func DecodeLZVNBuffer(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.DecodeLZVNBuffer(src, dst), nil
}

// EncodeZBMBuffer compresses src using the LZBITMAP container format into
// dst, returning the number of bytes written, or 0 on error or if dst is too
// small.
func EncodeZBMBuffer(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.EncodeZBMBuffer(src, dst), nil
}

// DecodeZBMBuffer decompresses LZBITMAP-encoded src into dst, returning the
// number of bytes written, or 0 on error or if dst is too small.
func DecodeZBMBuffer(src, dst []byte) (int, error) {
	lib, err := DefaultLibrary()
	if err != nil {
		return 0, err
	}
	return lib.DecodeZBMBuffer(src, dst), nil
}
