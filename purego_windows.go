//go:build windows && (amd64 || arm64)

package lzfse

import (
	"errors"
	"syscall"

	"github.com/ebitengine/purego"
)

func init() {
	libOpener = openLibrary
	symFinder = findSymbol
	libCloser = closeLibrary
}

func openLibrary(path string) (uintptr, error) {
	h, err := syscall.LoadLibrary(path)
	return uintptr(h), err
}

func findSymbol(handle uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(handle), name)
}

func closeLibrary(handle uintptr) { _ = syscall.FreeLibrary(syscall.Handle(handle)) }

func registerLibFunc(fptr any, addr uintptr) {
	purego.RegisterFunc(fptr, addr)
}

// loadDefaultLibrary loads the first library the search path produces.
func loadDefaultLibrary() (*Library, error) {
	paths, err := defaultSearchPaths()
	if err != nil {
		return nil, err
	}
	var errs []error
	for _, p := range paths {
		lib, err := NewLibrary(p)
		if err == nil {
			return lib, nil
		}
		errs = append(errs, err)
	}
	return nil, errors.Join(errs...)
}
