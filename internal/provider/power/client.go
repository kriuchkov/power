package power

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"power/internal/protomsg"
)

const ClientTimeout = 2 * time.Second

var ErrUnknownError = errors.New("unkonwn error")

// hasher is interface for solving the []byte hash
type hasher func(raw []byte) (response []byte, err error)

type Client struct {
	conn       net.Conn
	udpServer  *net.UDPAddr
	protocoler protomsg.ProtocolHandler
	protocol   protomsg.Protocoler
	dialTCP    handlerTCP
	solveHash  hasher
}

// GetMessage returns data on a body message
func (c *Client) GetMessage(ctx context.Context) (response []byte, err error) {
	verify, err := c.protocol.MessageUDP(c.udpServer, &protomsg.Message{
		Command: protomsg.CommandTypeVerify,
	})
	if err != nil {
		return response, err
	}

	hash, err := c.solveHash(verify.Data())
	if err != nil {
		return response, fmt.Errorf("[requestMessage] couldn't solve hash: %w", err)
	}

	tcp, err := c.protocol.MessageUDP(c.udpServer, &protomsg.Message{
		Command: protomsg.CommandTypeConnect,
		Body:    hash,
	})
	if err != nil {
		return response, err
	}

	conn, err := c.dialTCP(tcp.Data())
	if err != nil {
		return response, err
	}
	return c.handleTCPconnection(ctx, conn)
}

func (c *Client) handleTCPconnection(ctx context.Context, conn net.Conn) (response []byte, err error) {
	tcpProtocol := c.protocoler(ctx, conn)
	defer func() {
		tcpProtocol.Close()
	}()

	tcpProtocol.Send(conn.RemoteAddr(), &protomsg.Message{Command: protomsg.CommandTypeMsg})
	for {
		select {
		case <-ctx.Done():
			tcpProtocol.Send(conn.RemoteAddr(), &protomsg.Message{Command: protomsg.CommandTypeClose})
			return
		case <-time.After(ClientTimeout):
			tcpProtocol.Send(conn.RemoteAddr(), &protomsg.Message{Command: protomsg.CommandTypeClose})
			return
		case msg := <-tcpProtocol.Receiver():
			switch msg.Command {
			case protomsg.CommandTypeMsg:
				return msg.Data(), err
			case protomsg.CommandTypeClose:
				return nil, ErrUnknownError
			}
		}
	}
}

// Close sends a close message to the protocol
func (c *Client) Close() {
	c.protocol.Close()
}

// New returns a new client instance
func New(ctx context.Context, conn net.Conn, serv *net.UDPAddr, hash hasher, options ...clientHandler) *Client {
	cl := &Client{conn: conn, solveHash: hash, udpServer: serv}
	cl.protocoler = func(ctx context.Context, conn net.Conn) protomsg.Protocoler {
		return protomsg.New(ctx, conn, ClientTimeout)
	}
	cl.dialTCP = func(addr []byte) (net.Conn, error) {
		return net.Dial("tcp", string(addr))
	}

	for _, opts := range options {
		opts(cl)
	}
	cl.protocol = cl.protocoler(ctx, conn)
	return cl
}
