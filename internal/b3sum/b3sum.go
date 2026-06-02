// Package b3sum implements the b3sum command: BLAKE3 checksums of files and
// stdin, with a verify (--check) mode. Output is compatible with the reference
// b3sum: "<hex>  <name>", two spaces between digest and filename.
package b3sum

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-compressions/blake3"
)

// Run executes the b3sum CLI with the given args (excluding the program name).
// It reads stdin from in and writes to out/errw, returning a process exit code:
// 0 on success, 1 on a hash/verify/IO failure, 2 on a usage error.
func Run(args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("b3sum", flag.ContinueOnError)
	fs.SetOutput(errw)
	length := fs.Int("length", 32, "number of output bytes")
	lengthL := fs.Int("l", 0, "alias for --length (>= 1)")
	check := fs.Bool("check", false, "read checksums from the files and verify them")
	checkC := fs.Bool("c", false, "alias for --check")
	noNames := fs.Bool("no-names", false, "omit filenames from the output")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *lengthL != 0 {
		*length = *lengthL
	}
	if *length < 1 {
		fmt.Fprintln(errw, "b3sum: --length must be >= 1")
		return 2
	}

	files := fs.Args()
	if len(files) == 0 {
		files = []string{"-"}
	}

	if *check || *checkC {
		return runCheck(files, in, out, errw)
	}
	return runSum(files, *length, *noNames, in, out, errw)
}

// runSum hashes each file (or stdin for "-") and prints "<hex>  <name>".
func runSum(files []string, length int, noNames bool, in io.Reader, out, errw io.Writer) int {
	code := 0
	for _, name := range files {
		digest, err := hashOne(name, length, in)
		if err != nil {
			fmt.Fprintf(errw, "b3sum: %s: %v\n", name, err)
			code = 1
			continue
		}
		if noNames {
			fmt.Fprintf(out, "%s\n", digest)
		} else {
			fmt.Fprintf(out, "%s  %s\n", digest, name)
		}
	}
	return code
}

// runCheck reads "<hex>  <name>" lines from each checksum file (or stdin) and
// verifies them, printing "<name>: OK" or "<name>: FAILED".
func runCheck(files []string, in io.Reader, out, errw io.Writer) int {
	code := 0
	for _, cf := range files {
		var r io.Reader
		if cf == "-" {
			r = in
		} else {
			f, err := os.Open(cf)
			if err != nil {
				fmt.Fprintf(errw, "b3sum: %s: %v\n", cf, err)
				code = 1
				continue
			}
			defer f.Close()
			r = f
		}
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), "\r")
			if line == "" {
				continue
			}
			want, name, ok := parseCheckLine(line)
			if !ok {
				fmt.Fprintf(errw, "b3sum: malformed checksum line: %q\n", line)
				code = 1
				continue
			}
			got, err := hashOne(name, len(want)/2, in)
			if err != nil {
				fmt.Fprintf(errw, "b3sum: %s: %v\n", name, err)
				fmt.Fprintf(out, "%s: FAILED\n", name)
				code = 1
				continue
			}
			if strings.EqualFold(got, want) {
				fmt.Fprintf(out, "%s: OK\n", name)
			} else {
				fmt.Fprintf(out, "%s: FAILED\n", name)
				code = 1
			}
		}
	}
	return code
}

// parseCheckLine splits "<hex>  <name>" into the lowercase-or-uppercase hex
// digest and the filename. It reports ok=false if the line has no separator or
// a non-hex / odd-length digest.
func parseCheckLine(line string) (hexDigest, name string, ok bool) {
	i := strings.IndexByte(line, ' ')
	if i <= 0 {
		return "", "", false
	}
	hexDigest = line[:i]
	name = strings.TrimLeft(line[i:], " ")
	if name == "" {
		return "", "", false
	}
	if _, err := hex.DecodeString(hexDigest); err != nil {
		return "", "", false
	}
	return hexDigest, name, true
}

// hashOne reads name (or stdin for "-") fully and returns its BLAKE3 digest of
// the given byte length, hex-encoded. Reading the whole input lets the one-shot
// Sum256/Sum512 path hash large files with the multi-core/SIMD kernels.
func hashOne(name string, length int, in io.Reader) (string, error) {
	var data []byte
	var err error
	if name == "-" {
		data, err = io.ReadAll(in)
	} else {
		data, err = os.ReadFile(name)
	}
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest(data, length)), nil
}

// digest returns the BLAKE3 digest of data truncated/extended to length bytes.
// length<=32 and 33..64 use the parallel one-shot Sum256/Sum512; longer lengths
// use the streaming extendable-output Hasher.
func digest(data []byte, length int) []byte {
	switch {
	case length <= 32:
		d := blake3.Sum256(data)
		return d[:length]
	case length <= 64:
		d := blake3.Sum512(data)
		return d[:length]
	default:
		h := blake3.New()
		h.Write(data)
		out := make([]byte, length)
		h.Digest(out)
		return out
	}
}
