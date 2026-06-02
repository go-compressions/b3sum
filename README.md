# b3sum

[![ci](https://github.com/go-compressions/b3sum/actions/workflows/ci.yml/badge.svg)](https://github.com/go-compressions/b3sum/actions/workflows/ci.yml)
![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-compressions/b3sum.svg)](https://pkg.go.dev/github.com/go-compressions/b3sum)

Pure-Go, **cgo-free** `b3sum` — print and verify **BLAKE3** checksums of files
and stdin. Output is compatible with the reference [b3sum]: `<hex>  <name>`.

Built on [go-compressions/blake3](https://github.com/go-compressions/blake3). A
single static binary, no C toolchain.

```console
$ b3sum file.iso
e3b0c442...  file.iso

$ b3sum *.tar.gz > SUMS
$ b3sum --check SUMS
archive-1.tar.gz: OK
archive-2.tar.gz: OK

$ printf abc | b3sum --no-names
6437b3ac38465133ffb63b75273a8db548c558465d79db03fd359c6cd5bd9d85
```

## Usage

```
b3sum [--length N] [--no-names] [FILE...]   # hash files (or stdin with - / none)
b3sum --check [FILE...]                     # verify checksums read from FILE(s)
```

- `--length N`, `-l N` — output N bytes (default 32).
- `--check`, `-c` — verify `<hex>  <name>` lines; prints `<name>: OK` / `FAILED`,
  exits non-zero on any mismatch.
- `--no-names` — print only the digest.

## Performance / experimental SIMD

`b3sum` hashes each file in one shot, so it rides the underlying library's
multi-core kernels. The library also has an **experimental, cgo-free SIMD**
path (Go's `simd/archsimd` intrinsics) that helps on large files. To build an
optimized binary for testing:

```sh
GOEXPERIMENT=simd go build ./cmd/b3sum      # Go 1.26+ amd64 (AVX2), Go 1.27+ arm64 (NEON)
```

The default release is the portable pure-Go (scalar) build; an
`…-simd-experimental` amd64 binary is published alongside it for those who want
to test the SIMD path. The SIMD path is bit-identical to scalar (verified
against the official BLAKE3 vectors) and falls back to scalar on CPUs without
the required instructions and for small inputs.

## License

BSD-3-Clause. See [LICENSE](LICENSE).

[b3sum]: https://github.com/BLAKE3-team/BLAKE3
