package pow

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
)

const PowDigestLength = 20

type Pow struct {
	difficulty int
}

func NewPow(difficulty int) *Pow {
	return &Pow{difficulty: difficulty}
}

func (p *Pow) GenerateHash(msg []byte, nonce int) []byte {
	data := fmt.Sprintf("%s:%d", msg, nonce)
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

func (p *Pow) IsValidHash(hash []byte, byteIndex int, byteValue byte) bool {
	if len(hash) < p.difficulty {
		return false
	}

	//nolint:intrange // we are sure that p.difficulty is in the range of hash length
	for i := 0; i < p.difficulty; i++ {
		if hash[i] != '0' {
			return false
		}
	}
	return hash[byteIndex] == byteValue
}

func (p *Pow) GetClientConditions(clientAddr net.Addr) (int, byte) {
	ip := clientAddr.String()
	byteIndex := int(ip[0]) % 32
	byteValue := ip[len(ip)-1]
	return byteIndex, byteValue
}

func (p *Pow) FindNonce(ctx context.Context, hash []byte, byteIndex int, byteValue byte) int {
	var nonce = 0
	for {
		select {
		case <-ctx.Done():
			return -1
		default:
			clientHash := p.GenerateHash(hash, nonce)
			if p.IsValidHash(clientHash, byteIndex, byteValue) {
				return nonce
			}
			nonce++
		}
	}
}
