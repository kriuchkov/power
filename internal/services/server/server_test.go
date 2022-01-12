package server

import (
	"context"
	"net"
	"os"
	"testing"

	"power/internal/protomsg"
	ptcMocks "power/internal/protomsg/mocks"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

var (
	testBody = []byte("test")
	testAddr = net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
)

type testCacher struct{}

func (t *testCacher) Set(key, value interface{}) bool {
	return false
}

func (t *testCacher) Get(key interface{}) (interface{}, bool) {
	return nil, false
}

func (t *testCacher) Purge() {}

type testListner struct {
	net.Listener
	conn net.Conn
	addr *net.TCPAddr
}

func (t *testListner) Accept() (net.Conn, error) {
	return t.conn, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (t *testListner) Close() error {
	return t.conn.Close()
}

func (t *testListner) Addr() net.Addr {
	return &testAddr
}

func TestMain(m *testing.M) {
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetLevel(log.DebugLevel)
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestPowServerListen(t *testing.T) {
	defer goleak.VerifyNone(t)

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
		DoAndReturn(func(c net.Addr, m *protomsg.Message) { sender <- m })

	mockPtc.EXPECT().Receiver().AnyTimes().DoAndReturn(func() chan *protomsg.Message {
		return receiver
	})
	mockPtc.EXPECT().Close().AnyTimes().DoAndReturn(func() {
		cancel()
	})

	serverPip, clientPipe := net.Pipe()
	defer serverPip.Close()
	defer clientPipe.Close()

	addrs := []*net.TCPAddr{&testAddr, &testAddr, &testAddr}
	serv, err := New(serverPip, &testCacher{}, addrs,
		WithHandler(func() *protomsg.Message {
			return &protomsg.Message{ToAddr: serverPip.LocalAddr(), Body: testBody}
		}),
		WithProtocol(func(_ context.Context, _ net.Conn) protomsg.Protocoler {
			return mockPtc
		}),
		WithListenTCP(func(addr *net.TCPAddr) (net.Listener, error) {
			return &testListner{addr: addr, conn: clientPipe}, nil
		}),
		WithCheckVerify(func(msg *protomsg.Message) bool {
			return true
		}),
	)
	if err != nil {
		t.Errorf("couldn't create a new server instance %s", err.Error())
	}

	go serv.Listen(ctx)
	// the client sends empty a verification message
	receiver <- &protomsg.Message{ToAddr: serverPip.LocalAddr(), Command: protomsg.CommandTypeVerify}
	// get a hash message
	msg := <-sender
	// send a hash message
	receiver <- &protomsg.Message{ToAddr: serverPip.LocalAddr(), Command: protomsg.CommandTypeConnect, Body: msg.Data()}
	// got tcp connection
	msg = <-sender
	assert.Equal(t, string(msg.Body), testAddr.String())
}

func TestHandleConnection(t *testing.T) {
	defer goleak.VerifyNone(t)

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
		DoAndReturn(func(c net.Addr, m *protomsg.Message) { sender <- m })

	mockPtc.EXPECT().
		Receiver().
		AnyTimes().
		DoAndReturn(func() chan *protomsg.Message { return receiver })

	mockPtc.EXPECT().
		Close().
		AnyTimes().
		DoAndReturn(func() { cancel() })

	serv := &powServer{}
	serv.tcpAddresses = newPowTCPList(nil)
	serv.protocoler = func(_ context.Context, _ net.Conn) protomsg.Protocoler {
		return mockPtc
	}

	serverPip, clientPipe := net.Pipe()
	serv.bodyHandler = func() *protomsg.Message {
		return &protomsg.Message{ToAddr: serverPip.LocalAddr(), Body: testBody}
	}

	tln := &testListner{addr: &testAddr, conn: serverPip}
	go serv.handleConnection(ctx, tln, &testAddr)
	// the client sends a cobmand to get a message
	receiver <- &protomsg.Message{ToAddr: serverPip.LocalAddr(), Command: protomsg.CommandTypeMsg}
	// get a message
	msg := <-sender
	assert.Equal(t, msg.Body, testBody)

	serverPip.Close()
	clientPipe.Close()
}
