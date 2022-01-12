package protomsg

import (
	"context"
	"net"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestProtoMsg(t *testing.T) {
	defer goleak.VerifyNone(t)
	log.SetLevel(log.DebugLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, server := net.Pipe()
	protocol := New(ctx, client, 1*time.Second)

	testMessages := map[*Message]*Message{
		{
			ToAddr:  client.LocalAddr(),
			Command: CommandTypeVerify,
			Body:    []byte("test_verify")}: {Body: []byte("test_verify")},
	}

	for req, resp := range testMessages {
		go func() {
			_, err := read(server, make([]byte, ReadBuffer))
			assert.Nil(t, err)

			data, err := req.Marshal()
			assert.Nil(t, err)
			_, _ = write(server, data)
		}()

		respMsg, err := protocol.Message(resp)
		assert.Nil(t, err)
		assert.Equal(t, respMsg, req)
	}
	protocol.Close()
	client.Close()
	server.Close()
}
