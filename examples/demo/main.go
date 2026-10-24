// Command demo shows the lzfse package: it compresses and decompresses a
// file with LZFSE and prints the sizes.
//
//	go run ./examples/demo [file]
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bitxeno/lzfse-go"
)

func main() {
	path := "testdata/dec.bin"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	src, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	enc, err := lzfse.EncodeBuffer(src)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}
	dec, err := lzfse.DecodeBuffer(enc)
	if err != nil {
		log.Fatalf("decode: %v", err)
	}

	fmt.Printf("%s: %d bytes -> %d bytes -> %d bytes, roundtrip ok: %v\n",
		path, len(src), len(enc), len(dec), string(dec) == string(src))
}
