package pages

type Bitset struct {
	Bits []uint64
}

func NewBitset() *Bitset {
	return &Bitset{make([]uint64, 32)}
}

func (b *Bitset) Set(n int) {
	b.Bits[n>>6] |= 1 << (n & 63)
}

func (b Bitset) Has(n int) bool {
	return (b.Bits[n>>6]>>(n&63))&1 != 0
}
