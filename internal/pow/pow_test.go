package pow_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"testing"

	"github.com/kriuchkov/power/internal/pow"
	"github.com/stretchr/testify/require"
)

func TestGenerateHash(t *testing.T) {
	t.Parallel()

	p := pow.NewPow(4)

	tests := []struct {
		msg      []byte
		nonce    int
		expected [32]byte
	}{
		{
			msg:      []byte("test pow"),
			nonce:    0,
			expected: sha256.Sum256([]byte("test pow:0")),
		},
		{
			msg:      []byte(""),
			nonce:    999,
			expected: sha256.Sum256([]byte(":999")),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("msg=%s,once=%d", tt.msg, tt.nonce), func(t *testing.T) {
			t.Parallel()
			hash := p.GenerateHash(tt.msg, tt.nonce)
			require.Equal(t, tt.expected[:], hash)
		})
	}
}

func TestIsValidHash(t *testing.T) {
	t.Parallel()

	p := pow.NewPow(4)

	tests := []struct {
		hash      []byte
		byteIndex int
		byteValue byte
		expected  bool
	}{
		{
			hash:      []byte("0000abcd"),
			byteIndex: 4,
			byteValue: 'a',
			expected:  true,
		},
		{
			hash:      []byte("0000abcd"),
			byteIndex: 4,
			byteValue: 'b',
			expected:  false,
		},
		{
			hash:      []byte("000abcd"),
			byteIndex: 3,
			byteValue: 'a',
			expected:  false,
		},
		{
			hash:      []byte("0000abcd"),
			byteIndex: 5,
			byteValue: 'c',
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("hash=%s,byteIndex=%d,byteValue=%c", tt.hash, tt.byteIndex, tt.byteValue), func(t *testing.T) {
			t.Parallel()

			valid := p.IsValidHash(tt.hash, tt.byteIndex, tt.byteValue)
			require.Equal(t, tt.expected, valid)
		})
	}
}

func TestGetClientConditions(t *testing.T) {
	t.Parallel()

	p := pow.NewPow(4)

	tests := []struct {
		clientAddr    net.Addr
		expectedIndex int
		expectedValue byte
	}{
		{
			clientAddr:    &mockAddr{addr: "192.168.1.1"},
			expectedIndex: int('1') % 32,
			expectedValue: '1',
		},
		{
			clientAddr:    &mockAddr{addr: "127.0.0.1"},
			expectedIndex: int('1') % 32,
			expectedValue: '1',
		},
		{
			clientAddr:    &mockAddr{addr: "255.255.255.255"},
			expectedIndex: int('2') % 32,
			expectedValue: '5',
		},
	}

	for _, tt := range tests {
		t.Run(tt.clientAddr.String(), func(t *testing.T) {
			t.Parallel()

			byteIndex, byteValue := p.GetClientConditions(tt.clientAddr)
			require.Equal(t, tt.expectedIndex, byteIndex)
			require.Equal(t, tt.expectedValue, byteValue)
		})
	}
}

func TestFindNonce(t *testing.T) {
	t.Parallel()

	p := pow.NewPow(1)

	tests := []struct {
		hash      []byte
		nonce     int
		byteIndex int
		byteValue byte
	}{
		{
			hash:      []byte("0000abcd"),
			byteIndex: 4,
			byteValue: 'a',
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("hash=%s,byteIndex=%d,byteValue=%c", tt.hash, tt.byteIndex, tt.byteValue), func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			nonce := p.FindNonce(ctx, tt.hash, tt.byteIndex, tt.byteValue)
			clientHash := p.GenerateHash(tt.hash, nonce)

			valid := p.IsValidHash(clientHash, tt.byteIndex, tt.byteValue)
			require.True(t, valid)
		})
	}
}

type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.addr
}
