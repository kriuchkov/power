//nolint:testpackage //it's internal tests
package common

import (
	"fmt"
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		hash      []byte
		byteIndex int
		byteValue byte
	}{
		{
			name:      "valid message",
			body:      []byte(fmt.Sprintf("%s|%d|%d", []byte("hash"), 1, 'a')),
			hash:      []byte("hash"),
			byteIndex: 1,
			byteValue: 'a',
		},
		{
			name:      "invalid message",
			body:      []byte("invalid message"),
			hash:      []byte(nil),
			byteIndex: 0,
			byteValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, gotIndex, gotValue := SplitMessage(tt.body)
			require.Equal(t, tt.hash, gotHash)
			require.Equal(t, tt.byteIndex, gotIndex)
			require.Equal(t, tt.byteValue, gotValue)
		})
	}
}
