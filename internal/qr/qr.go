// Package qr renders a small QR code as inline SVG, entirely in-process, with
// no third-party dependency and no network call.
//
// It exists for one screen: TOTP enrollment. The whole reason to generate the
// image here, by hand, is ADR-0053 — a secret is held only where the act it
// authorises is performed. Encoding the otpauth:// URI ourselves keeps the
// secret off any third-party QR service AND out of any imported encoder; the
// bytes travel from the handler to the pixels without leaving this binary.
//
// Scope is deliberately narrow, matching its sole caller rather than the full
// ISO/IEC 18004 surface:
//
//   - Byte mode only. An otpauth:// URI is opaque bytes; the other segment
//     modes (numeric, alphanumeric, kanji) would never be selected for it.
//   - Error-correction level M (~15% recovery) — the level every authenticator
//     app scans without complaint, and the default such QRs are cut at.
//   - Versions 1..10 (up to 213 bytes of payload). An otpauth:// URI runs
//     ~100-150 bytes; 10 leaves ample headroom. A longer payload returns
//     ErrTooLong, and the caller keeps the secret text as the manual-entry
//     fallback — a code that will not fit degrades to typing, never to a broken
//     image.
//
// The encoder follows the standard construction (Reed-Solomon over GF(256),
// the eight data masks with penalty-scored selection, BCH format/version bits).
// qr_test.go pins it to independent Reed-Solomon and BCH known vectors and to a
// full-matrix golden, the same known-vector discipline the sibling auth package
// holds its TOTP code to. Its output was additionally cross-checked during
// development by decoding it with a ZXing-derived reader across versions 1-10 —
// that round-trip is what caught the alignment-placement bug the golden now
// guards, and is documented here rather than committed so the package keeps no
// third-party test dependency.
package qr

import (
	"errors"
	"fmt"
	"html"
	"strings"
)

// ErrTooLong is returned when the payload does not fit a version-10 byte-mode
// symbol at error-correction level M (213 bytes). The caller is expected to
// fall back to showing the payload as text.
var ErrTooLong = errors.New("qr: payload too long for a version-10 symbol")

// ecBlocks describes, for one version at error-correction level M, how the data
// codewords split into blocks and how many EC codewords each block carries.
// Group 2 blocks (when present) hold one more data codeword than group 1; a
// version with a uniform block layout leaves the group-2 fields zero.
type ecBlocks struct {
	totalDataCW  int   // data codewords across all blocks
	ecPerBlock   int   // EC codewords per block (same for every block)
	group1Blocks int   // number of group-1 blocks
	group1DataCW int   // data codewords in each group-1 block
	group2Blocks int   // number of group-2 blocks (0 when uniform)
	group2DataCW int   // data codewords in each group-2 block
	alignments   []int // alignment-pattern centre coordinates (nil for v1)
}

// versionsM is the error-correction characteristics table (ISO/IEC 18004
// Table 9) for level M, versions 1..10 — the only versions this encoder emits.
var versionsM = map[int]ecBlocks{
	1:  {16, 10, 1, 16, 0, 0, nil},
	2:  {28, 16, 1, 28, 0, 0, []int{6, 18}},
	3:  {44, 26, 1, 44, 0, 0, []int{6, 22}},
	4:  {64, 18, 2, 32, 0, 0, []int{6, 26}},
	5:  {86, 24, 2, 43, 0, 0, []int{6, 30}},
	6:  {108, 16, 4, 27, 0, 0, []int{6, 34}},
	7:  {124, 18, 4, 31, 0, 0, []int{6, 22, 38}},
	8:  {154, 22, 2, 38, 2, 39, []int{6, 24, 42}},
	9:  {182, 22, 3, 36, 2, 37, []int{6, 26, 46}},
	10: {216, 26, 4, 43, 1, 44, []int{6, 28, 50}},
}

const maxVersion = 10

// --- Galois field GF(256) with primitive polynomial 0x11d (x^8+x^4+x^3+x^2+1),
// generator alpha = 2. exp/log tables make Reed-Solomon multiplication a table
// lookup; exp is doubled so an index up to 508 never wraps.

var (
	gfExp [512]byte
	gfLog [256]byte
)

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfLog[byte(x)] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

// rsGenerator returns the degree-n Reed-Solomon generator polynomial,
// product over i of (x - alpha^i), coefficients high-degree first.
func rsGenerator(n int) []byte {
	g := []byte{1}
	for i := 0; i < n; i++ {
		next := make([]byte, len(g)+1)
		for j := 0; j < len(g); j++ {
			next[j] ^= g[j]                        // g[j] * x
			next[j+1] ^= gfMul(g[j], gfExp[i])     // g[j] * alpha^i
		}
		g = next
	}
	return g
}

// rsEncode returns the n Reed-Solomon EC codewords for data over GF(256).
func rsEncode(data []byte, n int) []byte {
	return rsRemainder(data, rsGenerator(n))
}

// rsRemainder returns the EC codewords for data given a precomputed generator
// polynomial: the remainder of data*x^deg divided by gen. Taking the generator
// as an argument lets interleave build it once per version rather than once per
// block.
func rsRemainder(data, gen []byte) []byte {
	n := len(gen) - 1
	res := make([]byte, len(data)+n)
	copy(res, data)
	for i := 0; i < len(data); i++ {
		coef := res[i]
		if coef == 0 {
			continue
		}
		for j := 0; j < len(gen); j++ {
			res[i+j] ^= gfMul(gen[j], coef)
		}
	}
	return res[len(data):]
}

// --- bit buffer: MSB-first bit accumulation into whole bytes.

type bitBuffer struct {
	bytes []byte
	nbits int
}

func (b *bitBuffer) append(val uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		if b.nbits%8 == 0 {
			b.bytes = append(b.bytes, 0)
		}
		if (val>>uint(i))&1 == 1 {
			b.bytes[b.nbits/8] |= byte(1 << uint(7-b.nbits%8))
		}
		b.nbits++
	}
}

// chooseVersion returns the smallest version 1..10 whose level-M byte-mode
// capacity holds dataLen bytes, or ok=false when none does.
func chooseVersion(dataLen int) (version int, ec ecBlocks, ok bool) {
	for v := 1; v <= maxVersion; v++ {
		blocks := versionsM[v]
		countBits := 8
		if v >= 10 {
			countBits = 16
		}
		if 4+countBits+dataLen*8 <= blocks.totalDataCW*8 {
			return v, blocks, true
		}
	}
	return 0, ecBlocks{}, false
}

// encodeData turns the payload into the version's full run of data codewords:
// the byte-mode segment (mode + count + bytes), a terminator, byte padding, and
// the alternating 0xEC/0x11 pad codewords that fill the capacity.
func encodeData(data []byte, version int, ec ecBlocks) []byte {
	var bb bitBuffer
	bb.append(0b0100, 4) // byte mode
	countBits := 8
	if version >= 10 {
		countBits = 16
	}
	bb.append(uint32(len(data)), countBits)
	for _, d := range data {
		bb.append(uint32(d), 8)
	}
	capacityBits := ec.totalDataCW * 8
	if term := capacityBits - bb.nbits; term > 0 {
		if term > 4 {
			term = 4
		}
		bb.append(0, term)
	}
	if pad := bb.nbits % 8; pad != 0 {
		bb.append(0, 8-pad)
	}
	for i := 0; len(bb.bytes) < ec.totalDataCW; i++ {
		if i%2 == 0 {
			bb.bytes = append(bb.bytes, 0xEC)
		} else {
			bb.bytes = append(bb.bytes, 0x11)
		}
	}
	return bb.bytes
}

// interleave splits the data codewords into their blocks, computes each block's
// EC codewords, and interleaves both by codeword index — the final codeword
// stream the symbol is filled with (ISO/IEC 18004 §7.6).
func interleave(dataCW []byte, ec ecBlocks) []byte {
	type block struct{ data, ecc []byte }
	var blocks []block
	gen := rsGenerator(ec.ecPerBlock) // identical for every block of this version
	pos := 0
	take := func(count, size int) {
		for i := 0; i < count; i++ {
			d := dataCW[pos : pos+size]
			pos += size
			blocks = append(blocks, block{d, rsRemainder(d, gen)})
		}
	}
	take(ec.group1Blocks, ec.group1DataCW)
	take(ec.group2Blocks, ec.group2DataCW)

	var out []byte
	maxData := ec.group1DataCW
	if ec.group2DataCW > maxData {
		maxData = ec.group2DataCW
	}
	for i := 0; i < maxData; i++ {
		for _, b := range blocks {
			if i < len(b.data) {
				out = append(out, b.data[i])
			}
		}
	}
	for i := 0; i < ec.ecPerBlock; i++ {
		for _, b := range blocks {
			out = append(out, b.ecc[i])
		}
	}
	return out
}

// --- matrix: the module grid plus a parallel mask of function (fixed) modules
// that carry no data and are never mask-inverted.

// Matrix is a rendered QR symbol: Size×Size modules, Dark(x,y) reporting
// whether a module is dark. It is returned by Encode for callers that want the
// raw grid; SVG is the usual entry point.
type Matrix struct {
	Size   int
	module [][]bool
	isFn   [][]bool
}

// Dark reports whether the module at column x, row y is dark.
func (m *Matrix) Dark(x, y int) bool { return m.module[y][x] }

func newMatrix(version int) *Matrix {
	size := version*4 + 17
	m := &Matrix{Size: size, module: make([][]bool, size), isFn: make([][]bool, size)}
	for i := range m.module {
		m.module[i] = make([]bool, size)
		m.isFn[i] = make([]bool, size)
	}
	return m
}

func (m *Matrix) set(x, y int, dark, fn bool) {
	if x < 0 || y < 0 || x >= m.Size || y >= m.Size {
		return
	}
	m.module[y][x] = dark
	if fn {
		m.isFn[y][x] = true
	}
}

func bit(v uint32, i int) bool { return (v>>uint(i))&1 == 1 }

// drawFunctionPatterns lays the finders, separators, timing patterns, the dark
// module, and the alignment patterns, and reserves the format/version regions.
func (m *Matrix) drawFunctionPatterns(version int, ec ecBlocks) {
	// Timing patterns (row 6 / column 6), laid first so finders overwrite the
	// overlap.
	for i := 0; i < m.Size; i++ {
		dark := i%2 == 0
		m.set(6, i, dark, true)
		m.set(i, 6, dark, true)
	}
	// Finder patterns with their one-module light separator, at three corners.
	m.drawFinder(3, 3)
	m.drawFinder(m.Size-4, 3)
	m.drawFinder(3, m.Size-4)
	// Alignment patterns at every centre pair, omitting only the three that
	// coincide with the finder patterns (the two remaining corners and the
	// top-left). The patterns that fall on the timing tracks ARE placed — from
	// version 7 up they overwrite the timing there, exactly as the standard
	// requires; skipping them (e.g. as "already a function module") leaves the
	// symbol unlocatable and only silently survives at versions 2-6, which
	// carry a single centre pattern.
	last := len(ec.alignments) - 1
	for i, ar := range ec.alignments {
		for j, ac := range ec.alignments {
			if (i == 0 && j == 0) || (i == 0 && j == last) || (i == last && j == 0) {
				continue
			}
			m.drawAlignment(ac, ar)
		}
	}
	// Reserve the format-info modules (written after masking) and, for v>=7,
	// the version-info modules.
	m.reserveFormat()
	if version >= 7 {
		m.reserveVersion()
	}
	// The always-dark module beside the bottom-left finder.
	m.set(8, m.Size-8, true, true)
}

// drawFinder draws a 7×7 finder centred at (cx,cy) plus its light separator.
func (m *Matrix) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x, y := cx+dx, cy+dy
			if x < 0 || y < 0 || x >= m.Size || y >= m.Size {
				continue
			}
			ax, ay := abs(dx), abs(dy)
			d := ax
			if ay > d {
				d = ay
			}
			// d==2 is the light ring, d==4 the separator; the rest is dark.
			m.set(x, y, d != 2 && d != 4, true)
		}
	}
}

// drawAlignment draws a 5×5 alignment pattern centred at (cx,cy).
func (m *Matrix) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			ax, ay := abs(dx), abs(dy)
			d := ax
			if ay > d {
				d = ay
			}
			m.set(cx+dx, cy+dy, d != 1, true)
		}
	}
}

// reserveFormat marks exactly the modules drawFormat later writes, so data
// placement skips them. It must not touch the two timing intersections
// (column 6, row 8) and (column 8, row 6), which stay part of the timing
// tracks — the standard format strips step over them.
func (m *Matrix) reserveFormat() {
	for i := 0; i <= 8; i++ {
		if i != 6 { // column 6 is the vertical timing track
			m.set(8, i, false, true)
		}
		if i != 6 { // row 6 is the horizontal timing track
			m.set(i, 8, false, true)
		}
	}
	for i := 0; i < 8; i++ {
		m.set(m.Size-1-i, 8, false, true)
	}
	for i := 0; i < 7; i++ {
		m.set(8, m.Size-1-i, false, true)
	}
}

func (m *Matrix) reserveVersion() {
	for i := 0; i < 18; i++ {
		a := m.Size - 11 + i%3
		b := i / 3
		m.set(b, a, false, true)
		m.set(a, b, false, true)
	}
}

// placeData walks the standard zigzag (two-column strips, upward then downward,
// right to left, skipping the vertical timing column) writing codeword bits
// MSB-first into every non-function module.
func (m *Matrix) placeData(codewords []byte) {
	i := 0
	total := len(codewords) * 8
	for right := m.Size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5 // step over the vertical timing column
		}
		for vert := 0; vert < m.Size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				upward := ((right + 1) & 2) == 0
				y := vert
				if upward {
					y = m.Size - 1 - vert
				}
				if m.isFn[y][x] || i >= total {
					continue
				}
				m.module[y][x] = bit(uint32(codewords[i>>3]), 7-(i&7))
				i++
			}
		}
	}
}

// maskCondition reports whether module (x,y) is inverted under the given mask.
func maskCondition(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return (x*y)%2+(x*y)%3 == 0
	case 6:
		return ((x*y)%2+(x*y)%3)%2 == 0
	case 7:
		return ((x+y)%2+(x*y)%3)%2 == 0
	}
	return false
}

// applyMask XORs the mask pattern into every non-function module. It is its own
// inverse, so calling it twice with the same mask restores the grid.
func (m *Matrix) applyMask(mask int) {
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; x++ {
			if !m.isFn[y][x] && maskCondition(mask, x, y) {
				m.module[y][x] = !m.module[y][x]
			}
		}
	}
}

// formatBits returns the 15-bit BCH-encoded format information for level M and
// the given mask: 5 data bits (level 00 + mask) with 10 BCH check bits, XORed
// with the 0x5412 constant mask (ISO/IEC 18004 §8.9).
func formatBits(mask int) uint32 {
	data := uint32(mask) // level M contributes 00 in the high two bits
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return (data<<10 | rem) ^ 0x5412
}

// versionBits returns the 18-bit BCH-encoded version information (6 data bits +
// 12 check bits) for version >= 7 (ISO/IEC 18004 §8.10).
func versionBits(version int) uint32 {
	rem := uint32(version)
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1f25)
	}
	return uint32(version)<<12 | rem
}

// drawFormat writes both copies of the 15-bit format information for the chosen
// mask, in the exact bit-to-module mapping of ISO/IEC 18004 §8.9 (bit 0 = LSB).
func (m *Matrix) drawFormat(mask int) {
	f := formatBits(mask)
	// First copy, around the top-left finder.
	for i := 0; i <= 5; i++ {
		m.set(8, i, bit(f, i), true) // column 8, rows 0..5
	}
	m.set(8, 7, bit(f, 6), true)
	m.set(8, 8, bit(f, 7), true)
	m.set(7, 8, bit(f, 8), true)
	for i := 9; i < 15; i++ {
		m.set(14-i, 8, bit(f, i), true) // row 8, columns 5..0
	}
	// Second copy, split across the top-right and bottom-left finders.
	for i := 0; i < 8; i++ {
		m.set(m.Size-1-i, 8, bit(f, i), true) // row 8, columns Size-1..Size-8
	}
	for i := 8; i < 15; i++ {
		m.set(8, m.Size-15+i, bit(f, i), true) // column 8, rows Size-7..Size-1
	}
}

// drawVersion writes both copies of the version information (v >= 7 only).
func (m *Matrix) drawVersion(version int) {
	if version < 7 {
		return
	}
	v := versionBits(version)
	for i := 0; i < 18; i++ {
		b := bit(v, i)
		a := m.Size - 11 + i%3
		c := i / 3
		m.set(c, a, b, true)
		m.set(a, c, b, true)
	}
}

// penalty scores a masked grid by the four ISO/IEC 18004 §8.8.2 rules; the mask
// with the lowest score is chosen. Lower is better.
func (m *Matrix) penalty() int {
	const (
		n1 = 3
		n2 = 3
		n3 = 40
		n4 = 10
	)
	score := 0
	size := m.Size

	// Rule 1: runs of five or more like-coloured modules in a row/column.
	for y := 0; y < size; y++ {
		runColor, run := m.module[y][0], 1
		for x := 1; x < size; x++ {
			if m.module[y][x] == runColor {
				run++
			} else {
				if run >= 5 {
					score += n1 + (run - 5)
				}
				runColor, run = m.module[y][x], 1
			}
		}
		if run >= 5 {
			score += n1 + (run - 5)
		}
	}
	for x := 0; x < size; x++ {
		runColor, run := m.module[0][x], 1
		for y := 1; y < size; y++ {
			if m.module[y][x] == runColor {
				run++
			} else {
				if run >= 5 {
					score += n1 + (run - 5)
				}
				runColor, run = m.module[y][x], 1
			}
		}
		if run >= 5 {
			score += n1 + (run - 5)
		}
	}

	// Rule 2: 2×2 blocks of one colour.
	for y := 0; y < size-1; y++ {
		for x := 0; x < size-1; x++ {
			c := m.module[y][x]
			if c == m.module[y][x+1] && c == m.module[y+1][x] && c == m.module[y+1][x+1] {
				score += n2
			}
		}
	}

	// Rule 3: the 1:1:3:1:1 finder-like pattern (with its light run) in any
	// row or column, either orientation.
	for y := 0; y < size; y++ {
		for x := 0; x <= size-11; x++ {
			if m.matchFinderRun(x, y, true) {
				score += n3
			}
		}
	}
	for x := 0; x < size; x++ {
		for y := 0; y <= size-11; y++ {
			if m.matchFinderRun(x, y, false) {
				score += n3
			}
		}
	}

	// Rule 4: deviation of the dark-module proportion from 50%.
	dark := 0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if m.module[y][x] {
				dark++
			}
		}
	}
	total := size * size
	k := (abs(dark*20-total*10) + total - 1) / total // ceil(|pct-50|/5)
	score += (k - 1) * n4

	return score
}

// finderRun is the 11-module 1:1:3:1:1-plus-light signature rule 3 penalises.
var finderRun = [11]bool{true, false, true, true, true, false, true, false, false, false, false}

func (m *Matrix) matchFinderRun(x, y int, horizontal bool) bool {
	forward, backward := true, true
	for i := 0; i < 11; i++ {
		var v bool
		if horizontal {
			v = m.module[y][x+i]
		} else {
			v = m.module[y+i][x]
		}
		if v != finderRun[i] {
			forward = false
		}
		if v != finderRun[10-i] {
			backward = false
		}
	}
	return forward || backward
}

// Encode builds the QR Matrix for data at error-correction level M, choosing
// the smallest fitting version (1..10) and the lowest-penalty mask. It returns
// ErrTooLong when data exceeds a version-10 symbol.
func Encode(data []byte) (*Matrix, error) {
	version, ec, ok := chooseVersion(len(data))
	if !ok {
		return nil, ErrTooLong
	}
	codewords := interleave(encodeData(data, version, ec), ec)

	m := newMatrix(version)
	m.drawFunctionPatterns(version, ec)
	m.drawVersion(version)
	m.placeData(codewords)

	bestMask, bestScore := 0, -1
	for mask := 0; mask < 8; mask++ {
		m.applyMask(mask)
		m.drawFormat(mask)
		if s := m.penalty(); bestScore < 0 || s < bestScore {
			bestScore, bestMask = s, mask
		}
		m.applyMask(mask) // undo
	}
	m.applyMask(bestMask)
	m.drawFormat(bestMask)
	return m, nil
}

// SVG encodes data and renders the symbol as a self-contained SVG string. The
// symbol is drawn as dark modules on an explicit white ground with a 4-module
// quiet zone, so it stays scannable regardless of the surrounding page theme.
// altText becomes the accessible name. It returns ErrTooLong when data does not
// fit a version-10 symbol.
func SVG(data []byte, altText string) (string, error) {
	m, err := Encode(data)
	if err != nil {
		return "", err
	}
	const quiet = 4
	dim := m.Size + 2*quiet

	// One subpath per horizontal run of dark modules (not per module), which
	// keeps the inline path compact.
	var path strings.Builder
	for y := 0; y < m.Size; y++ {
		for x := 0; x < m.Size; {
			if !m.module[y][x] {
				x++
				continue
			}
			run := 1
			for x+run < m.Size && m.module[y][x+run] {
				run++
			}
			fmt.Fprintf(&path, "M%d %dh%dv1h-%dz", x+quiet, y+quiet, run, run)
			x += run
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" height="100%%" shape-rendering="crispEdges" role="img" aria-label="%s">`,
		dim, dim, html.EscapeString(altText))
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, dim, dim)
	fmt.Fprintf(&b, `<path fill="#000000" d="%s"/>`, path.String())
	b.WriteString(`</svg>`)
	return b.String(), nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
