<p align="center"><img src="https://raw.githubusercontent.com/go-compressions/brand/main/social/go-compressions-b3sum.png" alt="go-compressions/b3sum" width="720"></p>

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

## Performance

`b3sum` hashes each file in one shot, so it rides the underlying library's
multi-core SIMD kernels. As of blake3 **v0.5.0** the **default build is
SIMD-accelerated on all six 64-bit targets** — cgo-free, no `GOEXPERIMENT` —
using [go-asmgen](https://github.com/go-asmgen/asmgen)-generated assembly for the
`mix4` round function: NEON (arm64), SSE2 (amd64), LSX (loong64), RVV (riscv64),
VSX (ppc64le) and the vector facility (s390x, big-endian). It is bit-identical to
the scalar path (verified against the official BLAKE3 vectors) and falls back to
scalar for small inputs. b3sum inherits all six arches with no code change.
**ppc64le is now natively measured on real POWER10** (GCC Compile Farm, VSX, Go
1.26.4, June 2026) — the underlying blake3 library's `mix4` runs ~4.5× scalar
there. **riscv64 is now natively measured too** on a real SpacemiT X60 (RVV 1.0,
a low-power in-order core — the only widely-available RVV silicon; GCC Compile
Farm, Go 1.26.4, June 2026), where the library's `mix4` runs ~2.9× scalar; an
out-of-order RVV core would likely do better. s390x stays qemu-validated for correctness only; native s390x throughput
is pending (no GitHub-hosted IBM Z runner). The blake3 library is additionally
build+test validated bit-exact on **ppc64 (big-endian)** on real POWER9 silicon
(generic/scalar fallback) — **six SIMD targets, validated on seven
architectures**. (These are library-level validations that b3sum inherits.)

Hashing a 1 GiB file (Apple Silicon, page cache warm):

| build | time | throughput | speedup |
|---|---|---|---|
| scalar (blake3 ≤ v0.3.0) | ~0.51 s | ~2.0 GB/s | 1.0x |
| **SIMD (blake3 ≥ v0.4.0, default)** | **~0.34 s** | **~3.0 GB/s** | **~1.5x** |

The underlying per-chunk kernel is ~3–4x faster than scalar; end-to-end b3sum
gains less because of file I/O and the tree/parent hashing. No flags needed —
just build or download the release binary.

### Alternative: archsimd build

The library also has a second SIMD implementation using Go's experimental
`simd/archsimd` intrinsics (amd64 AVX2 on Go 1.26+, arm64 on Go 1.27+). A
`…-simd-experimental` amd64 binary is published alongside the defaults for
comparison:

```sh
GOEXPERIMENT=simd go build ./cmd/b3sum
```

## License

BSD-3-Clause. See [LICENSE](LICENSE).

[b3sum]: https://github.com/BLAKE3-team/BLAKE3
