package b3sum

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-compressions/blake3"
)

// want computes the expected hex digest of data at the given byte length, the
// same way the command does, so tests assert against the library directly.
func want(data []byte, length int) string {
	return hex.EncodeToString(digest(data, length))
}

func run(t *testing.T, args []string, stdin string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = Run(args, strings.NewReader(stdin), &o, &e)
	return code, o.String(), e.String()
}

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSumStdin(t *testing.T) {
	code, out, _ := run(t, nil, "abc")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if want, got := want([]byte("abc"), 32)+"  -\n", out; got != want {
		t.Fatalf("out = %q, want %q", got, want)
	}
}

func TestSumFile(t *testing.T) {
	p := writeFile(t, "f.bin", []byte("hello world"))
	code, out, _ := run(t, []string{p}, "")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if !strings.HasPrefix(out, want([]byte("hello world"), 32)+"  "+p) {
		t.Fatalf("out = %q", out)
	}
}

func TestSumNoNames(t *testing.T) {
	code, out, _ := run(t, []string{"--no-names"}, "xyz")
	if code != 0 || out != want([]byte("xyz"), 32)+"\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestSumLengths(t *testing.T) {
	for _, n := range []int{16, 32, 48, 100} { // <=32, =32, 33-64, >64 branches
		_, out, _ := run(t, []string{"--length", itoa(n), "--no-names"}, "data")
		if got := strings.TrimSpace(out); got != want([]byte("data"), n) {
			t.Fatalf("length %d: out=%q want=%q", n, got, want([]byte("data"), n))
		}
	}
}

func TestLengthAlias(t *testing.T) {
	_, out, _ := run(t, []string{"-l", "16", "--no-names"}, "data")
	if got := strings.TrimSpace(out); got != want([]byte("data"), 16) {
		t.Fatalf("-l 16: out=%q want=%q", got, want([]byte("data"), 16))
	}
}

func TestSumLargeUsesParallel(t *testing.T) {
	// >1 chunk so the parallel/SIMD hashAll path is exercised; must match Sum256.
	data := bytes.Repeat([]byte{0xA5}, 200*1024)
	p := writeFile(t, "big.bin", data)
	_, out, _ := run(t, []string{"--no-names", p}, "")
	if got := strings.TrimSpace(out); got != want(data, 32) {
		t.Fatalf("large file digest mismatch")
	}
}

func TestSumMissingFile(t *testing.T) {
	code, out, errOut := run(t, []string{filepath.Join(t.TempDir(), "nope")}, "")
	if code != 1 || out != "" || !strings.Contains(errOut, "b3sum:") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestBadFlag(t *testing.T) {
	if code, _, _ := run(t, []string{"--nope"}, ""); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestBadLength(t *testing.T) {
	if code, _, errOut := run(t, []string{"--length", "0"}, ""); code != 2 || !strings.Contains(errOut, "length") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCheckOKAndFailed(t *testing.T) {
	good := writeFile(t, "good.bin", []byte("content"))
	bad := writeFile(t, "bad.bin", []byte("content"))
	lines := want([]byte("content"), 32) + "  " + good + "\n" +
		strings.Repeat("0", 64) + "  " + bad + "\n"
	cf := writeFile(t, "sums", []byte(lines))

	code, out, _ := run(t, []string{"--check", cf}, "")
	if code != 1 { // one OK, one FAILED -> overall failure
		t.Fatalf("code = %d", code)
	}
	if !strings.Contains(out, good+": OK") || !strings.Contains(out, bad+": FAILED") {
		t.Fatalf("out = %q", out)
	}
}

func TestCheckAliasAndStdin(t *testing.T) {
	f := writeFile(t, "f", []byte("z"))
	line := want([]byte("z"), 32) + "  " + f + "\n"
	code, out, _ := run(t, []string{"-c"}, line) // -c alias, checksum list from stdin
	if code != 0 || !strings.Contains(out, f+": OK") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestCheckMissingCheckFile(t *testing.T) {
	code, _, errOut := run(t, []string{"--check", filepath.Join(t.TempDir(), "nope")}, "")
	if code != 1 || !strings.Contains(errOut, "b3sum:") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCheckMalformedAndBlank(t *testing.T) {
	cf := writeFile(t, "sums", []byte("\nnoseparator\n"))
	code, _, errOut := run(t, []string{"--check", cf}, "")
	if code != 1 || !strings.Contains(errOut, "malformed") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCheckHashError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")
	line := strings.Repeat("ab", 32) + "  " + missing + "\n"
	cf := writeFile(t, "sums", []byte(line))
	code, out, errOut := run(t, []string{"--check", cf}, "")
	if code != 1 || !strings.Contains(out, "FAILED") || !strings.Contains(errOut, "b3sum:") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}
}

func TestCheckCarriageReturn(t *testing.T) {
	f := writeFile(t, "f", []byte("w"))
	cf := writeFile(t, "sums", []byte(want([]byte("w"), 32)+"  "+f+"\r\n"))
	if code, out, _ := run(t, []string{"--check", cf}, ""); code != 0 || !strings.Contains(out, f+": OK") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestParseCheckLine(t *testing.T) {
	h := strings.Repeat("ab", 32)
	cases := []struct {
		line     string
		wantOK   bool
		wantName string
	}{
		{h + "  file.txt", true, "file.txt"},
		{h + " file.txt", true, "file.txt"},
		{"noseparator", false, ""},
		{h + "   ", false, ""}, // empty name
		{"zz file", false, ""}, // non-hex digest
		{" leadingspace", false, ""},
	}
	for _, c := range cases {
		gotHex, gotName, ok := parseCheckLine(c.line)
		if ok != c.wantOK || (ok && (gotName != c.wantName || gotHex != h && c.line[0] != 'z')) {
			t.Errorf("parseCheckLine(%q) = (%q,%q,%v), want ok=%v name=%q", c.line, gotHex, gotName, ok, c.wantOK, c.wantName)
		}
	}
}

func TestKnownEmptyVector(t *testing.T) {
	// Official BLAKE3 empty-input digest.
	const empty = "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
	d := blake3.Sum256(nil)
	if hex.EncodeToString(d[:]) != empty {
		t.Fatalf("empty vector mismatch")
	}
	_, out, _ := run(t, []string{"--no-names"}, "")
	if strings.TrimSpace(out) != empty {
		t.Fatalf("b3sum of empty stdin = %q", out)
	}
}

// itoa avoids importing strconv just for the length-flag tests.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
