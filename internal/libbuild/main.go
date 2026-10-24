// Command libbuild builds the native lzfse shared libraries that this
// package embeds under libs/, using the vendored C sources in native/.
//
//	go run ./internal/libbuild            # build for this platform
//	go run ./internal/libbuild -all       # cross-compile every target with zig
//
// Zig (https://ziglang.org) is used as the cross compiler, as in
// github.com/egoist/mygo; when building only for the host, the system C
// compiler is used and zig is not required.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type target struct {
	// dir is the output directory under libs/, named after GOOS-GOARCH.
	dir string
	// zig is the zig target triple; empty means build with the host cc.
	zig string
	// file is the shared library file name on that platform.
	file string
}

var targets = []target{
	{"darwin-arm64", "aarch64-macos", "liblzfse-go.dylib"},
	{"darwin-amd64", "x86_64-macos", "liblzfse-go.dylib"},
	{"linux-amd64", "x86_64-linux-gnu", "liblzfse-go.so"},
	{"linux-arm64", "aarch64-linux-gnu", "liblzfse-go.so"},
	{"windows-amd64", "x86_64-windows-gnu", "lzfse-go.dll"},
	{"windows-arm64", "aarch64-windows-gnu", "lzfse-go.dll"},
}

// sources are the vendored C files compiled into the library: upstream lzfse
// (without its CLI main), libzbitmap, and the shim.
var sources = []string{
	"shim.c",
	"lzfse/lzfse_decode.c",
	"lzfse/lzfse_decode_base.c",
	"lzfse/lzfse_encode.c",
	"lzfse/lzfse_encode_base.c",
	"lzfse/lzfse_fse.c",
	"lzfse/lzvn_decode_base.c",
	"lzfse/lzvn_encode_base.c",
	"libzbitmap/libzbitmap.c",
}

func main() {
	all := flag.Bool("all", false, "cross-compile every supported target with zig")
	host := flag.Bool("host", false, "build only for this platform with the host C compiler")
	out := flag.String("out", "libs", "output directory")
	cc := flag.String("cc", "cc", "host C compiler")
	flag.Parse()

	if *all == *host {
		fmt.Fprintln(os.Stderr, "pass exactly one of -all or -host")
		os.Exit(2)
	}

	root, err := moduleRoot()
	if err != nil {
		fatal(err)
	}
	native := filepath.Join(root, "native")

	var selected []target
	if *all {
		selected = targets
	} else {
		dir := runtime.GOOS + "-" + runtime.GOARCH
		file := map[string]string{"windows": "lzfse-go.dll", "darwin": "liblzfse-go.dylib"}[runtime.GOOS]
		if file == "" {
			file = "liblzfse-go.so"
		}
		selected = []target{{dir: dir, file: file}}
	}

	for _, t := range selected {
		if err := build(native, filepath.Join(root, *out, t.dir), t, *all, *cc); err != nil {
			fatal(err)
		}
		fmt.Printf("built libs/%s/%s\n", t.dir, t.file)
	}
}

func build(native, outDir string, t target, useZig bool, hostCC string) error {
	var args []string
	cc := hostCC
	if useZig {
		cc = "zig"
		args = append(args, "cc", "-target", t.zig)
	}
	out := filepath.Join(outDir, t.file)
	args = append(args,
		"-O2", "-std=c99", "-fPIC", "-shared", "-s",
		"-Wall", "-Wno-unknown-pragmas", "-Wno-unused-variable",
		"-DLZFSE_DLL", "-DLZFSE_DLL_EXPORTS",
		"-I"+native,
		"-o", out,
	)
	for _, s := range sources {
		args = append(args, filepath.Join(native, s))
	}
	// Windows PE exports only what a .def file lists (the LZFSE_DLL_EXPORTS
	// dllexport annotations cover just the four lzfse.h functions); other
	// platforms export everything by default.
	if strings.HasSuffix(t.file, ".dll") {
		args = append(args, filepath.Join(native, "lzfse-go.def"))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command(cc, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", cc, strings.Join(args, " "), err)
	}
	// Remove everything but the library itself: zig leaves an import library
	// next to Windows DLLs, and stray files would end up embedded.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != t.file {
			if err := os.RemoveAll(filepath.Join(outDir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// moduleRoot walks up from the working directory to the checkout containing
// native/, so the tool runs the same from the repo root or from its own
// directory.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "native")); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot locate the module root (no native/ directory found)")
		}
		dir = parent
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "libbuild:", err)
	os.Exit(1)
}
