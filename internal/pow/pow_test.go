package pow

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPow(t *testing.T) {
	a := GenHash([]byte("test_01"), []byte("test_02"))
	assert.Greater(t, len(a), 0)

	response, err := SolveHash(a)
	assert.Nil(t, err)

	b := big.NewInt(0)
	b.SetBytes(response)
	assert.Equal(t, CheckHash(b), true)
}
