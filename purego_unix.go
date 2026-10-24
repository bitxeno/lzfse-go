//go:build (darwin || linux) && (amd64 || arm64)

package lzfse

import (
	"errors"
	"runtime"

	"github.com/ebitengine/purego"
)

func init() {
	libOpener = openLibrary
	symFinder = findSymbol
	libCloser = closeLibrary
}

func openLibrary(path string) (uintptr, error) {
	flags := purego.RTLD_NOW | purego.RTLD_LOCAL
	if runtime.GOOS == "linux" {
		// Keep our symbols private, but let the library resolve its own
		// symbols first should another loaded library export clashing ones.
		const rtldDeepBind = 0x00008
		flags |= rtldDeepBind
	}
	return purego.Dlopen(path, flags)
}

func findSymbol(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}

func closeLibrary(handle uintptr) { _ = purego.Dlclose(handle) }

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
