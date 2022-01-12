package power

import (
	"net"
	"power/internal/protomsg"
)

type clientHandler func(c *Client)
type handlerTCP func(addr []byte) (net.Conn, error)

//WithProtocol adds a protocol handler
func WithProtocol(protocolHandler protomsg.ProtocolHandler) clientHandler {
	return func(c *Client) {
		c.protocoler = protocolHandler
	}
}

//WithProtocol adds a tcp dialerq
func WithDialTCP(dialTCP handlerTCP) clientHandler {
	return func(c *Client) {
		c.dialTCP = dialTCP
	}
}
