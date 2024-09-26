//nolint:testpackage // ignore errcheck linter for this file
package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	clientmocks "github.com/kriuchkov/power/pkg/client/mocks"
	powerV1 "github.com/kriuchkov/protobuf/v1"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestClient_GetMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverResponse  func(t *testing.T) []byte
		solverFunc      func(t *testing.T) SolverHash
		expectedMessage []byte
		expectedErr     error
	}{
		{
			name: "success",
			serverResponse: func(_ *testing.T) []byte {
				var buf bytes.Buffer
				verifyMessage := &powerV1.Message{Command: powerV1.CommandType_Connect, Body: []byte("test")}
				verifyBytes, _ := proto.Marshal(verifyMessage)
				binary.Write(&buf, binary.BigEndian, int32(len(verifyBytes)))
				buf.Write(verifyBytes)

				contentMessage := &powerV1.Message{Command: powerV1.CommandType_Content, Body: []byte("response")}
				contentBytes, _ := proto.Marshal(contentMessage)
				binary.Write(&buf, binary.BigEndian, int32(len(contentBytes)))
				buf.Write(contentBytes)

				return buf.Bytes()
			},
			solverFunc: func(t *testing.T) SolverHash {
				mockSolver := clientmocks.NewMockSolverHash(t)
				mockSolver.EXPECT().FindNonce(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(123)
				return mockSolver
			},
			expectedMessage: []byte("response"),
			expectedErr:     nil,
		},
		{
			name: "error on connect message",
			serverResponse: func(_ *testing.T) []byte {
				var buf bytes.Buffer
				verifyMessage := &powerV1.Message{Command: powerV1.CommandType_Connect, Body: []byte("test")}
				verifyBytes, _ := proto.Marshal(verifyMessage)
				binary.Write(&buf, binary.BigEndian, int32(len(verifyBytes)))
				buf.Write(verifyBytes)

				return buf.Bytes()
			},
			solverFunc: func(t *testing.T) SolverHash {
				mockSolver := clientmocks.NewMockSolverHash(t)
				mockSolver.EXPECT().FindNonce(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(123)
				return mockSolver
			},
			expectedMessage: nil,
			expectedErr:     io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockConn := newMockConn(tt.serverResponse(t))

			cl := New(&Dependencies{ServerConn: &net.TCPConn{}, Hasher: tt.solverFunc(t)})
			cl.conn = mockConn

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			response, err := cl.GetMessage(ctx)
			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, tt.expectedMessage, response)
		})
	}
}

type mockConn struct {
	net.Conn
	readBuffer  *bytes.Buffer
	writeBuffer *bytes.Buffer
}

func newMockConn(response []byte) *mockConn {
	return &mockConn{
		readBuffer:  bytes.NewBuffer(response),
		writeBuffer: new(bytes.Buffer),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(_ time.Time) error {
	return nil
}
