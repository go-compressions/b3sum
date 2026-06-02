// Command b3sum prints or verifies BLAKE3 checksums of files and stdin.
//
//	b3sum [--length N] [--no-names] [FILE...]
//	b3sum --check [FILE...]
//
// Output is compatible with the reference b3sum. It is pure Go and cgo-free;
// build with GOEXPERIMENT=simd (Go 1.26+ amd64, Go 1.27+ arm64) to enable the
// experimental SIMD hashing kernels for large inputs.
package main

import (
	"os"

	"github.com/go-compressions/b3sum/internal/b3sum"
)

func main() {
	os.Exit(b3sum.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
