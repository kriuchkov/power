package server_test

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	server "github.com/kriuchkov/power/pkg/server"
	mocks "github.com/kriuchkov/power/pkg/server/mocks"

	powerV1 "github.com/kriuchkov/protobuf/v1"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMain(m *testing.M) {
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetLevel(log.DebugLevel)
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestHandleConnection(t *testing.T) {
	t.Parallel()

	type powGenerateHashCaller struct {
		callsCount int
		hash       []byte
	}

	type powIsValidHashCaller struct {
		callsCount int
		hash       []byte
		byteIndex  int
		byteValue  byte
		valid      bool
	}

	tests := []struct {
		name                  string
		address               string
		byteIndex             int
		byteValue             byte
		messageHandler        server.MessageHandler
		powGenerateHashCaller powGenerateHashCaller
		powIsValidHashCaller  powIsValidHashCaller
		inputMessage          *powerV1.Message
		responseMessage       *powerV1.Message
		expectError           error
	}{
		{
			name:                  "connect message",
			address:               ":19090",
			messageHandler:        func() []byte { return []byte("msg received") },
			byteIndex:             1,
			byteValue:             'a',
			powGenerateHashCaller: powGenerateHashCaller{callsCount: 1, hash: []byte("valid hash")},
			inputMessage:          &powerV1.Message{Command: powerV1.CommandType_Connect},
			responseMessage:       &powerV1.Message{Command: powerV1.CommandType_Connect, Body: []byte("valid hash|1|97")},
		},
		{
			name:                  "content message with valid hash",
			address:               ":19091",
			messageHandler:        func() []byte { return []byte("msg received") },
			byteIndex:             1,
			byteValue:             'a',
			powGenerateHashCaller: powGenerateHashCaller{callsCount: 1, hash: []byte("valid hash")},
			powIsValidHashCaller:  powIsValidHashCaller{callsCount: 1, hash: []byte("valid hash"), byteIndex: 1, byteValue: 'a', valid: true},
			inputMessage:          &powerV1.Message{Command: powerV1.CommandType_Content, Body: []byte("valid hash")},
			responseMessage:       &powerV1.Message{Command: powerV1.CommandType_Content, Body: []byte("msg received")},
		},
		{
			name:                  "content message with invalid hash",
			address:               ":19092",
			messageHandler:        func() []byte { return []byte("msg received") },
			byteIndex:             1,
			byteValue:             'a',
			powGenerateHashCaller: powGenerateHashCaller{callsCount: 1, hash: []byte("invalid hash")},
			powIsValidHashCaller:  powIsValidHashCaller{callsCount: 1, hash: []byte("invalid hash"), byteIndex: 1, byteValue: 'a', valid: false},
			inputMessage:          &powerV1.Message{Command: powerV1.CommandType_Content, Body: []byte("invalid hash")},
			responseMessage:       &powerV1.Message{Command: powerV1.CommandType_ErrInvalidHash},
		},
		{
			name:                  "close message",
			address:               ":19093",
			messageHandler:        func() []byte { return []byte("msg received") },
			powGenerateHashCaller: powGenerateHashCaller{callsCount: 1, hash: []byte("invalid hash")},
			inputMessage:          &powerV1.Message{Command: powerV1.CommandType_Close},
			expectError:           io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			powMock := mocks.NewMockPowHandler(t)

			powMock.EXPECT().GetClientConditions(mock.Anything).
				Return(tt.byteIndex, tt.byteValue).
				Times(1)

			if tt.powIsValidHashCaller.callsCount > 0 {
				powMock.EXPECT().IsValidHash(tt.powIsValidHashCaller.hash, tt.powIsValidHashCaller.byteIndex, tt.powIsValidHashCaller.byteValue).
					Return(tt.powIsValidHashCaller.valid).
					Times(tt.powIsValidHashCaller.callsCount)
			}

			if tt.powGenerateHashCaller.callsCount > 0 {
				powMock.EXPECT().GenerateHash(mock.Anything, mock.Anything).
					Return(tt.powGenerateHashCaller.hash).
					Times(tt.powGenerateHashCaller.callsCount + tt.powIsValidHashCaller.callsCount)
			}

			handler, err := server.New(&server.Dependencies{
				TCPAddress:     tt.address,
				MessageHandler: tt.messageHandler,
				PowHandler:     powMock,
			})
			require.NoError(t, err)

			go handler.Listen(ctx)

			conn, err := net.Dial("tcp", tt.address)
			require.NoError(t, err)
			defer conn.Close()

			msgBytes, err := proto.Marshal(tt.inputMessage)
			require.NoError(t, err)

			msgSize := int32(len(msgBytes))
			err = binary.Write(conn, binary.BigEndian, msgSize)
			require.NoError(t, err)

			_, err = conn.Write(msgBytes)
			require.NoError(t, err)

			// Expecting a response
			var responseSize int32
			err = binary.Read(conn, binary.BigEndian, &responseSize)
			if tt.expectError != nil {
				require.ErrorIs(t, err, tt.expectError)
				return
			}

			require.NoError(t, err)
			require.Greater(t, responseSize, int32(0))

			response := make([]byte, responseSize)
			_, err = conn.Read(response)
			require.NoError(t, err)

			var responseMessage powerV1.Message
			err = proto.Unmarshal(response, &responseMessage)
			require.NoError(t, err)

			require.Equal(t, tt.responseMessage.GetCommand(), responseMessage.GetCommand())
			require.Equal(t, tt.responseMessage.GetBody(), responseMessage.GetBody())
		})
	}
}
