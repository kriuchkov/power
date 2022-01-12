package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"time"

	"power/internal/pow"
	"power/internal/protomsg"

	log "github.com/sirupsen/logrus"
)

const (
	defaultTimeout = 2 * time.Second
)

var (
	ErrInvalidHash  = errors.New("hash isn't valid")
	ErrTCPListEmpty = errors.New("tcp list is empty")
	ErrTCPNotFound  = errors.New("tcp not found")
)

type cacher interface {
	Set(key, value interface{}) bool
	Get(key interface{}) (interface{}, bool)
	Purge()
}

// powServer is a structure with a pow server
type powServer struct {
	tcpAddresses *powTCPList
	listener     net.Conn
	listenTCP    ListenTCPHandler
	verify       VerifyHandler
	blocklist    cacher
	timeout      time.Duration
	bodyHandler  protomsg.MessageHandler
	protocoler   protomsg.ProtocolHandler
}

// Listen will set listener as a passive socket ready to accept incoming connection request
func (serv *powServer) Listen(ctx context.Context) {
	protocol := serv.protocoler(ctx, serv.listener)
	defer protocol.Close()

	for msg := range protocol.Receiver() {
		logger := log.WithFields(log.Fields{
			"to":      msg.Addr(),
			"from":    serv.listener.LocalAddr(),
			"command": msg.Command,
		})
		addr := msg.Addr()

		switch msg.Command {
		case protomsg.CommandTypeVerify:
			if _, exist := serv.blocklist.Get(addr); exist {
				continue
			}

			if msg.Command == protomsg.CommandTypeVerify && len(msg.Data()) == 0 {
				logger.Debug("[serverResponse]: send a hash message")
				nonce := big.NewInt(int64(rand.Intn(math.MaxInt64)) - 1)

				protocol.Send(msg.Addr(), &protomsg.Message{
					Command: protomsg.CommandTypeVerify,
					Body:    pow.GenHash(nonce.Bytes(), []byte("")),
				})
				continue
			}

			if !serv.verify(msg) {
				logger.Debug("[serverResponse]: a hash is invalid")
				serv.blocklist.Set(addr, struct{}{})
				continue
			}

		case protomsg.CommandTypeConnect:
			address, err := serv.tcpAddresses.Get()
			if err != nil {
				logger.WithError(err).Debug("[serverResponse]: tcpAddresses")
				protocol.Send(msg.Addr(), &protomsg.Message{
					Command: protomsg.CommandTypeClose,
					Body:    []byte(ErrTCPListEmpty.Error()),
				})
				continue
			}

			if !serv.verify(msg) {
				logger.Debug("[serverResponse]: a hash is invalid")
				serv.blocklist.Set(addr, struct{}{})
				continue
			}

			ln, _ := serv.listenTCP(address)
			logger.Debug("[serverResponse]: send a listenTCP message")

			protocol.Send(msg.Addr(), &protomsg.Message{
				Command: protomsg.CommandTypeConnect,
				Body:    []byte(ln.Addr().String()),
			})
			go serv.handleConnection(ctx, ln, address)
		case protomsg.CommandTypeClose:
			return
		}
	}
}

// handleConnection handles a incoming connection
func (serv *powServer) handleConnection(ctx context.Context, tcpListener net.Listener, addr *net.TCPAddr) {
	defer func() { tcpListener.Close(); serv.tcpAddresses.Free(addr) }()

	conn, err := tcpListener.Accept()
	if err != nil {
		log.WithError(err).Debug("[handleConnection] error")
		return
	}
	defer conn.Close()

	protocol := serv.protocoler(ctx, conn)
	defer protocol.Close()

	logger := log.WithField("session", conn.RemoteAddr())
	logger.WithFields(log.Fields{"ln": tcpListener, "addr": addr}).Debug("[handleConnection] handle a new connection")
	defer func() { logger.Debug("[handleConnection] handler is closed") }()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(serv.timeout):
			return
		case msg := <-protocol.Receiver():
			if msg == nil {
				continue
			}
			logger := log.WithFields(log.Fields{
				"t":    fmt.Sprintf("%T", conn),
				"to":   conn.RemoteAddr(),
				"from": conn.LocalAddr(),
				"c":    msg.Command,
			})
			switch msg.Command {
			case protomsg.CommandTypeMsg:
				logger.Debug("[handleConnection] send a body message")
				protocol.Send(conn.RemoteAddr(), serv.bodyHandler())
			case protomsg.CommandTypeClose:
				logger.Debug("[handleConnection] gotten a close message")
				protocol.Send(conn.RemoteAddr(), &protomsg.Message{Command: protomsg.CommandTypeClose})
				return
			default:
				continue
			}
		}
	}
}

// New returns a new powServer instance
func New(udp net.Conn, chacheList cacher, addrs []*net.TCPAddr, options ...ServeHandler) (*powServer, error) {
	log.Debugf("udp server: bound to addr: %s", udp.LocalAddr().String())
	pow := &powServer{
		listener:     udp,
		tcpAddresses: newPowTCPList(addrs),
		timeout:      defaultTimeout,
		blocklist:    chacheList,
		bodyHandler:  func() (resp *protomsg.Message) { return &protomsg.Message{} },
		verify:       checkVerify,
	}

	log.WithField("addrs", pow.tcpAddresses).Debug("With tcp addrs")

	pow.protocoler = func(ctx context.Context, conn net.Conn) protomsg.Protocoler {
		return protomsg.New(ctx, conn, pow.timeout)
	}

	pow.listenTCP = func(addr *net.TCPAddr) (net.Listener, error) {
		return net.ListenTCP("tcp", addr)
	}

	for _, opts := range options {
		opts(pow)
	}
	return pow, nil
}

func NewUDP(host, port string, chacheList cacher, addrs []*net.TCPAddr, options ...ServeHandler) (*powServer, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return New(listener, chacheList, addrs, options...)
}

func checkVerify(verifyMsg *protomsg.Message) bool {
	bigI := big.NewInt(0)
	bigI.SetBytes(verifyMsg.Data())
	return pow.CheckHash(bigI)
}
