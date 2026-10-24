package lzfse

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// The shared libraries built by native/libbuild are embedded so that the
// package is self-contained: on first use the copy for the current platform
// is extracted to the user cache directory and loaded from there.
//
//go:embed all:libs
var embeddedLibraries embed.FS

// platformDir is the directory under libs that native/libbuild fills for the
// current platform, e.g. "libs/darwin-arm64".
func platformDir() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}

// libraryFileName is the file name the native library carries on this
// platform inside libs and in the cache directory.
func libraryFileName() string {
	switch runtime.GOOS {
	case "windows":
		return "lzfse-go.dll"
	case "darwin":
		return "liblzfse-go.dylib"
	default:
		return "liblzfse-go.so"
	}
}

// userLibraryDir is where the embedded library is extracted to: the user
// cache directory keyed by the library version, e.g.
// ~/Library/Caches/lzfse-go/1.0.0 on macOS.
func userLibraryDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "lzfse-go", Version), nil
}

// extractEmbeddedLibrary writes the embedded library for the current platform
// to dir, returning the extracted file path. The file is replaced on every
// call, so a rebuilt library is picked up on the next run.
func extractEmbeddedLibrary(dir string) (string, error) {
	name := libraryFileName()
	embedded := "libs/" + platformDir() + "/" + name
	f, err := embeddedLibraries.Open(embedded)
	if err != nil {
		return "", fmt.Errorf("lzfse: no embedded library for %s (run 'go run ./internal/libbuild -host' to build one): %w", platformDir(), err)
	}
	defer f.Close()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dstPath := filepath.Join(dir, name)
	tmpPath := dstPath + ".tmp"
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, f); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := dst.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, dstPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return dstPath, nil
}

// defaultSearchPaths implements the package-level library search: the
// LZFSE_LIBRARY_PATH environment variable wins, then the embedded copy,
// extracted to the user cache directory on first use.
func defaultSearchPaths() ([]string, error) {
	if p := os.Getenv(envLibraryPath); p != "" {
		return []string{p}, nil
	}
	dir, err := userLibraryDir()
	if err != nil {
		return nil, err
	}
	// Always re-extract, so an embedded library rebuilt between runs is
	// picked up even at the same version.
	libPath, err := extractEmbeddedLibrary(dir)
	if err != nil {
		return nil, err
	}
	return []string{libPath}, nil
}
