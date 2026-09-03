package wildcarddiscrim

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// One label each, so a control falls off at the same encloser as the candidates (ADR-0069, #115).

const (
	RandomLabelCount     = 9
	StructuredLabelCount = 1
	LabelCount           = RandomLabelCount + StructuredLabelCount
	randomLabelLen       = 32
)

// RFC 5737 space parses just as RFC 1918 does, so this separates a parser from a wildcard.

var rfc5737Blocks = [3][3]int{
	{192, 0, 2},
	{198, 51, 100},
	{203, 0, 113},
}

// Labels reach no output, so a deterministic generator keeps the golden corpus byte-identical.

type LabelGen interface {
	Labels() []string
}

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
