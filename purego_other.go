//go:build !((darwin || linux || windows) && (amd64 || arm64))

package lzfse

func loadDefaultLibrary() (*Library, error) {
	return nil, ErrUnsupported
}

func registerLibFunc(fptr any, addr uintptr) {
	panic("lzfse: unsupported platform")
}
