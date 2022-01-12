//go:generate mockgen -source protomsg.go -destination ./mocks/protomsg.go -package mocks
package protomsg

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	powerV1 "github.com/kryuchkovnet/protobuf/v1"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	ReadBuffer int = 1024
)

var (
	ErrMessageTimeout   = errors.New("couldn't receive a messege in time")
	ErrUnknownCommand   = errors.New("unknown command")
	ErrIncorrectCommand = errors.New("incorrect command")
	ErrBufferEmpty      = errors.New("empty buffer")
)

// Protocoler is interface for protocl like ProtoMsg
type Protocoler interface {
	Message(req *Message) (*Message, error)
	MessageUDP(addr *net.UDPAddr, req *Message) (*Message, error)
	Send(addr net.Addr, req *Message)
	Receiver() chan *Message
	Close()
}

// ProtoMsgHandler is a function hadler for the protoMsg
type ProtoMsgHandler func(msg *ProtoMsg) error

// ProtocolHandler is a ctx and conn handler returing a new instance
type ProtocolHandler func(ctx context.Context, conn net.Conn) Protocoler

// ProtoMsg is handler working with I/O net.Conn recieving and sending messages through channels.
type ProtoMsg struct {
	closed       chan struct{}
	wg           *sync.WaitGroup
	receiver     chan *Message
	transmission chan *Message
	timeout      time.Duration
}

// Message sends a message and waits to receive message,
// if the message wasn't received the function will return an error
func (p *ProtoMsg) Message(req *Message) (*Message, error) {
	p.transmission <- req
	select {
	case <-time.After(p.timeout):
		return nil, ErrMessageTimeout
	case msg := <-p.receiver:
		return msg, nil
	}
}

// MessageUDP send a udo message and waits to receive message,
// if the message wasn't received the function will return an error
func (p *ProtoMsg) MessageUDP(addr *net.UDPAddr, req *Message) (*Message, error) {
	req.ToAddr = addr
	p.transmission <- req
	select {
	case <-time.After(p.timeout):
		return nil, ErrMessageTimeout
	case msg := <-p.receiver:
		return msg, nil
	}
}

// Send sends a message to the transmission channel
func (p *ProtoMsg) Send(addr net.Addr, req *Message) {
	req.ToAddr = addr
	p.transmission <- req
}

// Receiver returns the receiver channl
func (p *ProtoMsg) Receiver() chan *Message {
	return p.receiver
}

//read reads a byte message and sends to the receiver channel transforming to a message
func (p *ProtoMsg) read(ctx context.Context, conn net.Conn) {
	var (
		err  error
		n    int
		addr net.Addr
	)
	defer func() { log.WithError(err).Debug("[protoMsg.read] readed is closed") }()

	buf := make([]byte, ReadBuffer)
	loop := true
	go func(l *bool) {
		defer p.wg.Done()
		<-ctx.Done()
		log.Debug("[protoMsg.read] readed is being closing by context")
		*l = false
	}(&loop)

	for loop {
		switch v := conn.(type) {
		case *net.TCPConn:
			n, err = read(v, buf)
			addr = conn.RemoteAddr()
		case *net.UDPConn:
			n, addr, err = readfromURP(v, buf)
		default:
			n, err = read(v, buf)
			addr = conn.RemoteAddr()
			log.WithField("t", fmt.Sprintf("%T", v)).Debug("[protoMsg.read] unknown type")
		}

		if errors.Is(err, ErrBufferEmpty) {
			continue
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			return
		}

		protoMsg := powerV1.Message{}
		err = proto.Unmarshal(buf[0:n], &protoMsg)
		if err != nil {
			log.WithError(err).WithField("size", n).Info("[protoMsg.read]: couldn't Unmarshal a request message")
			continue
		}

		msg := &Message{ToAddr: addr, Command: CommandType(protoMsg.GetCommand()), Body: protoMsg.GetBody()}
		if msg.Command == 0 {
			continue
		}
		log.WithField("size", len(msg.Body)).
			WithField("command", msg.Command).
			WithField("t", fmt.Sprintf("%T", msg)).
			Debug("[protoMsg.read]: gotten a request message")
		p.receiver <- msg
	}
}

// write writes a message to conn
func (p *ProtoMsg) write(ctx context.Context, conn net.Conn) {
	defer func() { log.Debug("[protoMsg.write] writer is closed"); p.wg.Done() }()

	var (
		err     error
		n       int
		sendMsg []byte
	)
	for {
		select {
		case <-ctx.Done():
			log.Debug("[protoMsg.write] writer is being closing by context")
			return
		case msg := <-p.transmission:
			if msg == nil {
				continue
			}
			logger := log.WithFields(log.Fields{"to": msg.Addr(), "from": conn.LocalAddr()})

			sendMsg, err = msg.Marshal()
			logger = logger.
				WithField("type", fmt.Sprintf("%T", msg)).
				WithField("size", len(sendMsg)).
				WithField("c", msg.Command)

			logger.Debug("[protoMsg.write]: prepare to send a message")

			if err != nil {
				logger.WithError(err).Info("[protoMsg.write]: couldn't marshal a message")
				continue
			}

			switch v := conn.(type) {
			case *net.TCPConn:
				n, err = write(v, sendMsg)
			case *net.UDPConn:
				n, err = writeToUDP(v, msg.ToAddr.(*net.UDPAddr), sendMsg)
			default:
				n, err = write(v, sendMsg)
				log.WithField("t", fmt.Sprintf("%T", v)).Debug("[protoMsg.write] unknown type")
			}

			if err != nil {
				logger.WithError(err).Info("[protoMsg.write]: couldn't send a message")
				continue
			}
			logger.WithField("n", n).Debug("[protoMsg.write] a message was sened")
		}
	}
}

// Close closed protocol channels
func (p *ProtoMsg) Close() {
	p.closed <- struct{}{}
	<-p.closed
	close(p.closed)
}

// New returns a new instance protoMsg
func New(ctx context.Context, conn net.Conn, timeout time.Duration) *ProtoMsg {
	ctx, cancel := context.WithCancel(ctx)

	ptc := &ProtoMsg{
		wg:           &sync.WaitGroup{},
		closed:       make(chan struct{}),
		receiver:     make(chan *Message),
		transmission: make(chan *Message),
		timeout:      timeout,
	}

	ptc.wg.Add(2)
	go ptc.read(ctx, conn)
	go ptc.write(ctx, conn)

	go func() {
		<-ptc.closed
		cancel()
		ptc.wg.Wait()
		close(ptc.receiver)
		close(ptc.transmission)
		ptc.closed <- struct{}{}
	}()
	return ptc
}

func write(conn net.Conn, msg []byte) (int, error) {
	writer := bufio.NewWriter(conn)
	number, err := writer.Write(msg)
	if err == nil {
		err = writer.Flush()
	}
	return number, err
}

func writeToUDP(conn *net.UDPConn, addr *net.UDPAddr, msg []byte) (int, error) {
	return conn.WriteTo(msg, addr)
}

func read(conn net.Conn, buf []byte) (int, error) {
	n, err := conn.Read(buf)
	if err != nil {
		return n, err
	}
	if n == 0 || n == -1 {
		return n, ErrBufferEmpty
	}
	return n, nil
}

func readfromURP(conn *net.UDPConn, buf []byte) (int, *net.UDPAddr, error) {
	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return n, addr, err
	}
	if n == 0 {
		return n, addr, ErrBufferEmpty
	}
	return n, addr, nil
}
