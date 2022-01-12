package pow

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math"
	"math/big"
)

const PowDigestLength = 20

var ErrNotFoundSolution = errors.New("could not find a solution")

// GenHash merges data and prev as bytes using bytes.Join
// creating a sha256 hash from this merge
func GenHash(data, prev []byte) []byte {
	head := bytes.Join([][]byte{prev, data}, []byte{})
	h32 := sha256.Sum256(head)
	output := h32[:]
	return output
}

// SolveHash iterates until enough leading zeros has been found
func SolveHash(raw []byte) (response []byte, err error) {
	var nonce int64
	for nonce = 0; nonce < math.MaxInt64; nonce++ {
		bigI := big.NewInt(0)
		bigI.Add(bigI.SetBytes(raw), big.NewInt(nonce))

		hash := sha256.Sum256(bigI.Bytes())
		if CheckHash(bigI.SetBytes(hash[:])) {
			return hash[:], nil
		}
	}
	return response, ErrNotFoundSolution
}

// CheckHash checks the n first bits in sha1(hash.seed) for zeroes
// checking the target number (0x8000...) bigger than old
func CheckHash(hash *big.Int) bool {
	target := big.NewInt(1)
	target = target.Lsh(target, uint(256-PowDigestLength))
	return target.Cmp(hash) > 0
}
