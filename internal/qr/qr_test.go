package qr

import (
	"strings"
	"testing"
)

// TestReedSolomonKnownVector pins the GF(256) Reed-Solomon encoder to the
// canonical "HELLO WORLD" Version 1-M worked example (Thonky QR tutorial): 16
// data codewords produce these 10 error-correction codewords. This is the same
// known-vector discipline internal/auth holds its TOTP code to.
func TestReedSolomonKnownVector(t *testing.T) {
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	got := rsEncode(data, 10)
	if len(got) != len(want) {
		t.Fatalf("EC length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("EC codeword %d = %d, want %d (full %v)", i, got[i], want[i], got)
		}
	}
}

// TestFormatBits pins the BCH format-information encoder to the published
// level-M strings for all eight masks (Thonky format-information table).
func TestFormatBits(t *testing.T) {
	want := []uint32{
		0b101010000010010, // mask 0
		0b101000100100101, // mask 1
		0b101111001111100, // mask 2
		0b101101101001011, // mask 3
		0b100010111111001, // mask 4
		0b100000011001110, // mask 5
		0b100111110010111, // mask 6
		0b100101010100000, // mask 7
	}
	for mask, w := range want {
		if got := formatBits(mask); got != w {
			t.Errorf("formatBits(%d) = %015b, want %015b", mask, got, w)
		}
	}
}

// TestVersionBits pins the BCH version-information encoder to the published
// 18-bit strings for the versions that carry one (7..10).
func TestVersionBits(t *testing.T) {
	want := map[int]uint32{
		7:  0x07C94,
		8:  0x085BC,
		9:  0x09A99,
		10: 0x0A4D3,
	}
	for v, w := range want {
		if got := versionBits(v); got != w {
			t.Errorf("versionBits(%d) = %05X, want %05X", v, got, w)
		}
	}
}

// TestChooseVersion checks the smallest fitting level-M byte-mode version at a
// few capacity boundaries.
func TestChooseVersion(t *testing.T) {
	cases := []struct {
		n, version int
		ok         bool
	}{
		{1, 1, true},
		{14, 1, true},  // v1-M byte capacity
		{15, 2, true},  // spills to v2
		{213, 10, true}, // v10-M byte capacity
		{214, 0, false}, // beyond v10
	}
	for _, c := range cases {
		v, _, ok := chooseVersion(c.n)
		if ok != c.ok || (ok && v != c.version) {
			t.Errorf("chooseVersion(%d) = (v%d, %v), want (v%d, %v)", c.n, v, ok, c.version, c.ok)
		}
	}
}

// TestEncodeStructure checks the invariant function patterns of a rendered
// symbol: correct dimensions, the three finder patterns with their light rings,
// alternating timing tracks, and the always-dark module.
func TestEncodeStructure(t *testing.T) {
	m, err := Encode([]byte("otpauth://totp/Verge%20ASM:alice?secret=JBSWY3DPEHPK3PXP&issuer=Verge%20ASM&digits=6&period=30"))
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if (m.Size-17)%4 != 0 || m.Size < 21 {
		t.Fatalf("Size %d is not a valid QR dimension", m.Size)
	}

	finder := func(cx, cy int) {
		t.Helper()
		for dy := -3; dy <= 3; dy++ {
			for dx := -3; dx <= 3; dx++ {
				d := max(abs(dx), abs(dy))
				want := d != 2 // dark everywhere but the light ring at Chebyshev distance 2
				if m.module[cy+dy][cx+dx] != want {
					t.Fatalf("finder(%d,%d) module (%d,%d) = %v, want %v", cx, cy, cx+dx, cy+dy, m.module[cy+dy][cx+dx], want)
				}
			}
		}
	}
	finder(3, 3)
	finder(m.Size-4, 3)
	finder(3, m.Size-4)

	// Timing tracks alternate, dark at even coordinate.
	for i := 8; i < m.Size-8; i++ {
		if m.module[6][i] != (i%2 == 0) || m.module[i][6] != (i%2 == 0) {
			t.Fatalf("timing module at %d not alternating", i)
		}
	}
	// Always-dark module beside the bottom-left finder.
	if !m.module[m.Size-8][8] {
		t.Fatalf("dark module at (8,%d) is not set", m.Size-8)
	}
}

// TestVersion7AlignmentOnTimingTrack pins the alignment-placement rule that
// only silently held at versions 2-6: from version 7 up, alignment patterns
// centred on the timing tracks (here (col 22, row 6)) MUST be placed. A payload
// of ~108 bytes forces version 7. Omitting these patterns leaves the symbol
// undecodable, so this guards the regression the ZXing round-trip caught.
func TestVersion7AlignmentOnTimingTrack(t *testing.T) {
	m, err := Encode([]byte(strings.Repeat("a", 108)))
	if err != nil {
		t.Fatal(err)
	}
	if v := (m.Size - 17) / 4; v != 7 {
		t.Fatalf("payload landed on version %d, want 7", v)
	}
	// Alignment pattern centred at column 22, row 6: dark centre, a light ring
	// at Chebyshev distance 1, dark border at distance 2.
	cx, cy := 22, 6
	if !m.module[cy][cx] {
		t.Errorf("alignment centre (%d,%d) is not dark — pattern missing", cx, cy)
	}
	if m.module[cy][cx-1] || m.module[cy][cx+1] || m.module[cy-1][cx] || m.module[cy+1][cx] {
		t.Errorf("alignment ring around (%d,%d) is not light — pattern missing or malformed", cx, cy)
	}
}

// TestEncodeIsDeterministic guards against a masking/penalty non-determinism
// regression: the same payload must render identically every time.
func TestEncodeIsDeterministic(t *testing.T) {
	payload := []byte("otpauth://totp/Verge%20ASM:bob?secret=KVKFKRCPNZQUYMLXOVYDSQKJKZDTSRLD&issuer=Verge%20ASM&digits=6&period=30")
	a, err := Encode(payload)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if a.Size != b.Size {
		t.Fatalf("size differs: %d vs %d", a.Size, b.Size)
	}
	for y := 0; y < a.Size; y++ {
		for x := 0; x < a.Size; x++ {
			if a.module[y][x] != b.module[y][x] {
				t.Fatalf("non-deterministic module at (%d,%d)", x, y)
			}
		}
	}
}

func TestSVG(t *testing.T) {
	svg, err := SVG([]byte("otpauth://totp/Verge%20ASM:carol?secret=JBSWY3DPEHPK3PXP&issuer=Verge%20ASM&digits=6&period=30"), "TOTP enrollment QR")
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	for _, want := range []string{
		`<svg`,
		`xmlns="http://www.w3.org/2000/svg"`,
		`role="img"`,
		`aria-label="TOTP enrollment QR"`,
		`fill="#ffffff"`, // explicit light ground so it scans on any theme
		`<path fill="#000000"`,
		`</svg>`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q\n%s", want, svg)
		}
	}
	if strings.Contains(svg, "http://") && strings.Count(svg, "http://") > 1 {
		t.Errorf("SVG references an external URL beyond the SVG namespace")
	}
}

func TestSVGAltTextEscaped(t *testing.T) {
	svg, err := SVG([]byte("otpauth://x"), `a"b<c&d`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, `aria-label="a&quot;b&lt;c&amp;d"`) {
		t.Errorf("alt text not escaped: %s", svg)
	}
}

func TestEncodeTooLong(t *testing.T) {
	if _, err := Encode(make([]byte, 214)); err != ErrTooLong {
		t.Fatalf("Encode(214 bytes) err = %v, want ErrTooLong", err)
	}
	if _, err := Encode(make([]byte, 213)); err != nil {
		t.Fatalf("Encode(213 bytes) err = %v, want nil", err)
	}
}
