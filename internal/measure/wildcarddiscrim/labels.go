package wildcarddiscrim

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// The control-label set is 9 random labels plus 1 structured label, each exactly
// one label (ADR-0069, raised to 9 by #115). A control label is one label so it
// falls off the tree at the same closest encloser the candidates do; a
// multi-label label measures a different, deeper wildcard.
const (
	RandomLabelCount     = 9
	StructuredLabelCount = 1
	LabelCount           = RandomLabelCount + StructuredLabelCount
	randomLabelLen       = 32
)

// rfc5737Blocks are the three documentation address ranges the structured label
// draws its octets from. RFC 5737 space is decoded by an address-parsing
// authority exactly as RFC 1918 space is, so the structured label separates a
// parser from a wildcard while keeping *long random*'s defence against accidental
// existence.
var rfc5737Blocks = [3][3]int{
	{192, 0, 2},    // 192.0.2.0/24
	{198, 51, 100}, // 198.51.100.0/24
	{203, 0, 113},  // 203.0.113.0/24
}

// LabelGen produces one Batch's control-label set. The labels are drawn per
// Batch as independent samples; they never appear in the leaf's output (a control
// label is an input to the decision and a value on no timeline), so a
// deterministic generator produces byte-identical observations for the golden
// corpus while production draws from crypto/rand.
type LabelGen interface {
	Labels() []string
}

// CryptoLabels is the production LabelGen: crypto/rand random labels plus a
// structured label over a random RFC 5737 documentation address.
type CryptoLabels struct{}

func (CryptoLabels) Labels() []string {
	out := make([]string, 0, LabelCount)
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < RandomLabelCount; i++ {
		b := make([]byte, randomLabelLen)
		for j := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			b[j] = alphabet[n.Int64()]
		}
		out = append(out, string(b))
	}
	for i := 0; i < StructuredLabelCount; i++ {
		blk := rfc5737Blocks[randInt(len(rfc5737Blocks))]
		last := randInt(256)
		out = append(out, fmt.Sprintf("%d-%d-%d-%d", blk[0], blk[1], blk[2], last))
	}
	return out
}

func randInt(n int) int {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}
