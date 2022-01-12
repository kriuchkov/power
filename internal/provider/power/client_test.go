package power

import (
	"context"
	"net"
	"testing"

	"power/internal/protomsg"
	ptcMocks "power/internal/protomsg/mocks"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var testBody = []byte("test")

func TestClient(t *testing.T) {
	defer goleak.VerifyNone(t)
	log.SetLevel(log.DebugLevel)

	//done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sender := make(chan *protomsg.Message)
	receiver := make(chan *protomsg.Message)
	defer func() { close(sender); close(receiver) }()

	mockPtc := ptcMocks.NewMockProtocoler(ctrl)

	mockPtc.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(c net.Addr, m *protomsg.Message) { m.ToAddr = c; sender <- m })

	mockPtc.EXPECT().
		Receiver().
		AnyTimes().
		DoAndReturn(func() chan *protomsg.Message { return receiver })

	mockPtc.EXPECT().
		Close().
		AnyTimes().
		DoAndReturn(func() { cancel() })

	mockPtc.EXPECT().
		Message(gomock.Any()).
		AnyTimes().
		DoAndReturn(func(a *protomsg.Message) (*protomsg.Message, error) {
			sender <- a
			return <-receiver, nil
		})
	mockPtc.EXPECT().
		MessageUDP(gomock.Any(), gomock.Any()).
		AnyTimes().
		DoAndReturn(func(c *net.UDPAddr, a *protomsg.Message) (*protomsg.Message, error) {
			a.ToAddr = c
			sender <- a
			return <-receiver, nil
		})

	clientPipe, _ := net.Pipe()
	defer clientPipe.Close()

	client := New(
		ctx,
		clientPipe,
		nil,
		testSolveHash,
		WithProtocol(func(_ context.Context, _ net.Conn) protomsg.Protocoler {
			return mockPtc
		}),
		WithDialTCP(func(_ []byte) (net.Conn, error) {
			return clientPipe, nil
		}),
	)
	go func() {
		_, err := client.GetMessage(ctx)
		assert.Nil(t, err)
	}()
	// get a empty verefication message
	msg := <-sender
	assert.Equal(t, msg.Command, protomsg.CommandTypeVerify)
	// send a hash message
	receiver <- &protomsg.Message{Command: protomsg.CommandTypeConnect, Body: testBody}
	// get a tcp connection
	msg = <-sender
	assert.Equal(t, msg.Command, protomsg.CommandTypeConnect)
	// get a connect message
	receiver <- &protomsg.Message{Command: protomsg.CommandTypeConnect, Body: testBody}
	// Handle a tcp connection
	msg = <-sender
	assert.Equal(t, msg.Command, protomsg.CommandTypeMsg)
	// get a body message
	receiver <- &protomsg.Message{Command: protomsg.CommandTypeMsg, Body: testBody}
	client.Close()
}

func testSolveHash(raw []byte) (response []byte, err error) {
	return []byte("test"), nil
}
