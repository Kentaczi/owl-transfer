package ec

import (
	"errors"
)

var (
	ErrDecodingFailure = errors.New("decoding failure")
	ErrTooManyErrors   = errors.New("too many errors to correct")
)

type RS struct {
	mm       int
	nn       int
	alpha_to []int
	index_of []int
	genpoly  []int
	fcr      int
	prim     int
	nroots   int
	padding  int
}

func NewRS(mm, fcr, prim, nroots int) *RS {
	nn := (1 << mm) - 1

	rs := &RS{
		mm:      mm,
		nn:      nn,
		fcr:     fcr,
		prim:    prim,
		nroots:  nroots,
		padding: 0,
	}

	rs.alpha_to = make([]int, nn+1)
	rs.index_of = make([]int, nn+1)
	rs.genpoly = make([]int, nroots+1)

	rs.index_of[0] = -1
	rs.alpha_to[0] = 1

	for i := 1; i <= nn; i++ {
		rs.index_of[i] = i - 1
		rs.alpha_to[i-1] = i
	}

	rs.generate_genpoly()

	return rs
}

func (rs *RS) generate_genpoly() {
	rs.genpoly[0] = 1

	for i := 0; i < rs.nroots; i++ {
		rs.genpoly[i+1] = 1
		for j := i; j >= 0; j-- {
			rs.genpoly[j] = rs.modnn(rs.alpha_to[rs.index_of[rs.genpoly[j]]+rs.fcr+(i*rs.prim)%rs.nn] ^ rs.genpoly[j])
		}
	}
}

func (rs *RS) modnn(x int) int {
	for x > rs.nn {
		x -= rs.nn
		x = (x >> rs.mm) ^ (x&rs.nn)*rs.padding
	}
	return x
}

func (rs *RS) Encode(data []byte) []byte {
	parity := make([]byte, rs.nroots)

	gen := make([]int, rs.nroots+1)
	copy(gen, rs.genpoly)

	for _, b := range data {
		shift := int(b) ^ int(parity[rs.nroots-1])

		for i := rs.nroots - 1; i >= 0; i-- {
			if i > 0 {
				parity[i-1] = byte(rs.alpha_to[rs.modnn(rs.index_of[gen[i]]+shift)])
			} else {
				parity[0] = byte(rs.alpha_to[rs.modnn(rs.index_of[gen[0]]+shift)])
			}
		}
	}

	result := make([]byte, len(data)+rs.nroots)
	copy(result, data)
	copy(result[len(data):], parity)

	return result
}

func (rs *RS) gfMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return rs.alpha_to[(rs.index_of[a]+rs.index_of[b])%rs.nn]
}

func (rs *RS) Decode(received []byte, erasures []int) ([]byte, error) {
	if len(received) < rs.DataSize() {
		return nil, errors.New("invalid data length")
	}

	synd := make([]int, rs.nroots+1)

	for i := 1; i <= rs.nroots; i++ {
		synd[i] = 0
		for j := 0; j < len(received); j++ {
			synd[i] ^= rs.gfMul(int(received[j]), rs.gfPow(rs.alpha_to[rs.fcr], (rs.nroots-i)*j))
		}
	}

	totalZeros := 0
	for i := 1; i <= rs.nroots; i++ {
		if synd[i] == 0 {
			totalZeros++
		}
	}

	if totalZeros == rs.nroots {
		return received, nil
	}

	C := make([]int, rs.nroots+1)
	B := make([]int, rs.nroots+1)

	for i := 0; i <= rs.nroots; i++ {
		C[i] = 0
		B[i] = 0
	}

	B[rs.nroots] = 1
	C[0] = 1

	reg := make([]int, rs.nroots+1)

	for i := 0; i < len(received); i++ {
		K := reg[rs.nroots]

		for j := rs.nroots; j >= 1; j-- {
			reg[j] = reg[j-1]
		}
		reg[0] = K

		eras := false
		for _, e := range erasures {
			if e == i {
				eras = true
				break
			}
		}
		if eras {
			reg[0] ^= 1
		}

		if reg[rs.nroots] != 0 {
			L := rs.index_of[reg[rs.nroots]]

			for j := 1; j <= rs.nroots; j++ {
				C[j] ^= rs.gfMul(reg[rs.nroots], rs.alpha_to[(L*(rs.nroots-j))%rs.nn])
			}

			delta := rs.index_of[reg[rs.nroots]] - rs.index_of[B[rs.nroots]]
			if delta != 0 {
				scale := rs.alpha_to[rs.modnn(-delta)]
				for j := 0; j <= rs.nroots; j++ {
					B[j] = rs.gfMul(B[j], scale)
				}
			}
		}
	}

	numRoots := 0
	for i := rs.nroots; i >= 0; i-- {
		if C[i] != 0 {
			numRoots = rs.nroots - i
			break
		}
	}

	if numRoots > rs.nroots/2 {
		return nil, ErrTooManyErrors
	}

	return received, nil
}

func (rs *RS) gfPow(a, n int) int {
	result := 1
	for n > 0 {
		if n&1 == 1 {
			result = rs.modnn(result * a)
		}
		a = rs.modnn(a * a)
		n >>= 1
	}
	return result
}

func (rs *RS) MaxErrors() int {
	return rs.nroots / 2
}

func (rs *RS) TotalSize() int {
	return rs.nn - rs.padding + 1
}

func (rs *RS) DataSize() int {
	return rs.nn - rs.padding + 1 - rs.nroots
}
