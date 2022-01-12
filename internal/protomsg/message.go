package protomsg

import (
	"net"

	powerV1 "github.com/kryuchkovnet/protobuf/v1"
	"google.golang.org/protobuf/proto"
)

// messager is interface for a message
type Messager interface {
	Marshal() ([]byte, error)
	Data() []byte
}

// MessageHandler is type for handling messages
type MessageHandler func() (resp *Message)

type CommandType int32

const (
	CommandTypeVerify  CommandType = 1
	CommandTypeConnect CommandType = 2
	CommandTypeMsg     CommandType = 3
	CommandTypeClose   CommandType = 4
)

// Message is general structure for sending and recieving messages
type Message struct {
	ToAddr  net.Addr
	Command CommandType
	Body    []byte
}

func (m *Message) Addr() net.Addr {
	return m.ToAddr
}

func (m *Message) Data() []byte {
	return m.Body
}

func (m *Message) Marshal() ([]byte, error) {
	return proto.Marshal(&powerV1.Message{
		Command: powerV1.CommandType(m.Command),
		Body:    m.Body,
	})
}
