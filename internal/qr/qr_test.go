package qr

import (
	"strings"
	"testing"
)

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

func TestChooseVersion(t *testing.T) {
	cases := []struct {
		n, version int
		ok         bool
	}{
		{1, 1, true},
		{14, 1, true},   // v1-M byte capacity
		{15, 2, true},   // spills to v2
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
	if !m.module[m.Size-8][8] {
		t.Fatalf("dark module at (8,%d) is not set", m.Size-8)
	}
}

// goldenPayload and goldenMatrix pin one full symbol end-to-end. The matrix is
// the version-6 rendering of goldenPayload, captured from this encoder and then
// confirmed to decode back to goldenPayload by a ZXing-derived reader (see the
// package doc). Unlike the sub-block vectors, it exercises the whole pipeline at
// once — interleaving, zigzag placement, mask selection, and format bits — so a
// regression in any of them that still passes the structural checks (an
// unscannable but well-formed symbol) fails here.
const goldenPayload = "otpauth://totp/Verge%20ASM:golden?secret=JBSWY3DPEHPK3PXP&issuer=Verge%20ASM&digits=6&period=30"

var goldenMatrix = []string{
	"11111110110110101111111110101011101111111",
	"10000010111011101010100101101000001000001",
	"10111010001010101100100101100001101011101",
	"10111010111000010000010001011100001011101",
	"10111010010010000100001110010001001011101",
	"10000010000010011001110011001111101000001",
	"11111110101010101010101010101010101111111",
	"00000000100011110110001010101111100000000",
	"10110111010110000011010000000101001001011",
	"10100001011110101100100101001011011111101",
	"10011111011001101100100100000110110111010",
	"01001100011011100001000101100010000100011",
	"10100011100100010000100010111101110001111",
	"11111000001000010110101110110101001001110",
	"10100111001001001011101100000011111000111",
	"10011100100000000010010100011000100111001",
	"11111010111010001010101011010010010010000",
	"10110000100111110000011101010110110101000",
	"00011011010010100010101000010010100110000",
	"10000100000110010110010110111111110101101",
	"10011011011111111000000011000111111110000",
	"11001000000110110000110001001111111011011",
	"00011110111110000000101111001010001000110",
	"00001101101101101011000101100010111110010",
	"11100010101011101000100000111101100001101",
	"00000001000101110101100011011111111101101",
	"11010010001101001001011010000101011001011",
	"00011000110100001011011000001000011011001",
	"01111011010111111110101110011011110111010",
	"00000100100000000101101001011010110100110",
	"10001010011000000011010001110110000011110",
	"00011000111000100111001100001100100011111",
	"01111110010010010110000000100101111111100",
	"00000000111110111110111111011101100010101",
	"11111110110110011010000111001011101010100",
	"10000010110011010100000010000010100010011",
	"10111010010001101000110000100100111111001",
	"10111010111011001000001110111101010110110",
	"10111010101111011011000000101011000100111",
	"10000010011011001100111110000011101010010",
	"11111110111010101100010101000010101101010",
}

func TestGoldenMatrix(t *testing.T) {
	m, err := Encode([]byte(goldenPayload))
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != len(goldenMatrix) {
		t.Fatalf("size %d, want %d", m.Size, len(goldenMatrix))
	}
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; x++ {
			want := goldenMatrix[y][x] == '1'
			if m.module[y][x] != want {
				t.Fatalf("module (%d,%d) = %v, want %v — encoder output drifted from the golden symbol", x, y, m.module[y][x], want)
			}
		}
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
	if !strings.Contains(svg, `aria-label="a&#34;b&lt;c&amp;d"`) {
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
