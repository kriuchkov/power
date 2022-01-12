package server

import (
	"net"
	"time"

	"power/internal/protomsg"
)

type ServeHandler func(pow *powServer)

type ListenTCPHandler func(addr *net.TCPAddr) (net.Listener, error)
type VerifyHandler func(msg *protomsg.Message) bool

//WithHandler adds a protomsg message handler
func WithHandler(msgHandler protomsg.MessageHandler) ServeHandler {
	return func(pow *powServer) {
		pow.bodyHandler = msgHandler
	}
}

//WithTimeout adds a timeout handler
func WithTimeout(d time.Duration) ServeHandler {
	return func(pow *powServer) {
		pow.timeout = d
	}
}

//WithProtocol adds a protocol handler
func WithProtocol(protocolHandler protomsg.ProtocolHandler) ServeHandler {
	return func(pow *powServer) {
		pow.protocoler = protocolHandler
	}
}

//WithListenTCP adds a tcp handler
func WithListenTCP(handler ListenTCPHandler) ServeHandler {
	return func(pow *powServer) {
		pow.listenTCP = handler
	}
}

// WithCheckVerify adds handler for verification
func WithCheckVerify(check VerifyHandler) ServeHandler {
	return func(pow *powServer) {
		pow.verify = check
	}
}
