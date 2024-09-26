package client

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/kriuchkov/power/pkg/common"

	"github.com/go-faster/errors"
	"github.com/go-playground/validator/v10"
	powerV1 "github.com/kriuchkov/protobuf/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const DefaultClientTimeout = 2 * time.Second

var (
	ErrWrongCommand = errors.New("wrong command")
	ErrInvalidHash  = errors.New("found invalid hash")
)

type SolverHash interface {
	FindNonce(ctx context.Context, hash []byte, byteIndex int, byteValue byte) int
}

type Dependencies struct {
	ServerConn net.Conn   `validate:"required"`
	Hasher     SolverHash `validate:"required"`
}

func (d *Dependencies) SetDefaults() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(d); err != nil {
		panic(err)
	}
}

type Client struct {
	conn   net.Conn
	solver SolverHash
}

func New(deps *Dependencies) *Client {
	deps.SetDefaults()
	return &Client{conn: deps.ServerConn, solver: deps.Hasher}
}

// GetMessage returns data on a body message.
//
//nolint:funlen,nonamedreturns // it's a client method
func (c *Client) GetMessage(ctx context.Context) (response []byte, err error) {
	err = c.conn.SetWriteDeadline(time.Now().Add(DefaultClientTimeout))
	if err != nil {
		return response, errors.Wrap(err, "set write deadline")
	}

	message := &powerV1.Message{Command: powerV1.CommandType_Connect}
	bytesMessage, err := proto.Marshal(message)
	if err != nil {
		return response, errors.Wrap(err, "marshal message")
	}

	msgSize := int32(len(bytesMessage))
	err = binary.Write(c.conn, binary.BigEndian, msgSize)
	if err != nil {
		return response, errors.Wrap(err, "send a connect message size")
	}

	_, err = c.conn.Write(bytesMessage)
	if err != nil {
		return response, errors.Wrap(err, "send a connect message")
	}

	log.WithFields(log.Fields{"size": msgSize, "command": message.GetCommand()}).
		Debug("send a connect message")

	err = c.conn.SetReadDeadline(time.Now().Add(DefaultClientTimeout))
	if err != nil {
		return response, errors.Wrap(err, "set read deadline")
	}

	err = binary.Read(c.conn, binary.BigEndian, &msgSize)
	if err != nil {
		return response, errors.Wrap(err, "read a verify message size")
	}

	verifyRawMessage := make([]byte, msgSize)
	_, err = c.conn.Read(verifyRawMessage)
	if err != nil {
		return response, errors.Wrap(err, "read a verify message")
	}

	var verifyMessage powerV1.Message
	err = proto.Unmarshal(verifyRawMessage, &verifyMessage)
	if err != nil {
		return response, errors.Wrap(err, "unmarshal response message")
	}

	if verifyMessage.GetCommand() != powerV1.CommandType_Connect {
		return response, ErrWrongCommand
	}

	serverHash, byteIndex, byteValue := common.SplitMessage(verifyMessage.GetBody())
	log.WithFields(log.Fields{"s": msgSize, "c": verifyMessage.GetCommand(), "i": byteIndex, "bv": byteValue}).
		Debug("read a verify message")

	// third step: find a nonce
	foundNonce := c.solver.FindNonce(ctx, serverHash, byteIndex, byteValue)

	log.WithFields(log.Fields{"nonce": foundNonce}).Debug("found nonce")

	message = &powerV1.Message{Command: powerV1.CommandType_Content, Body: []byte(strconv.Itoa(foundNonce))}

	bytesMessage, err = proto.Marshal(message)
	if err != nil {
		return response, errors.Wrap(err, "marshal message")
	}

	msgSize = int32(len(bytesMessage))
	err = binary.Write(c.conn, binary.BigEndian, msgSize)
	if err != nil {
		return response, errors.Wrap(err, "send a hash message size")
	}

	_, err = c.conn.Write(bytesMessage)
	if err != nil {
		return response, errors.Wrap(err, "send a hash message")
	}

	log.WithFields(log.Fields{"size": msgSize, "command": message.GetCommand()}).
		Debug("send a hash message")

	// fifth step: read a message
	err = c.conn.SetReadDeadline(time.Now().Add(DefaultClientTimeout))
	if err != nil {
		return response, errors.Wrap(err, "set read deadline")
	}

	err = binary.Read(c.conn, binary.BigEndian, &msgSize)
	if err != nil {
		return response, errors.Wrap(err, "read a message size")
	}

	contentBytes := make([]byte, msgSize)

	_, err = c.conn.Read(contentBytes)
	if err != nil {
		return response, errors.Wrap(err, "read a message")
	}

	var contentMessage powerV1.Message
	err = proto.Unmarshal(contentBytes, &contentMessage)
	if err != nil {
		return response, errors.Wrap(err, "unmarshal response message")
	}

	//nolint:exhaustive //ok
	switch contentMessage.GetCommand() {
	case powerV1.CommandType_ErrInvalidHash:
		return response, ErrInvalidHash
	case powerV1.CommandType_Content:
		return contentMessage.GetBody(), nil
	default:
		return response, ErrWrongCommand
	}
}
