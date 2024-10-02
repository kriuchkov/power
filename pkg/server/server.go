package server

import (
	"context"
	"encoding/binary"
	"io"
	"math/rand"
	"net"

	"github.com/kriuchkov/power/internal/pow"
	"github.com/kriuchkov/power/pkg/common"

	powerV1 "github.com/kriuchkov/protobuf/v1"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/go-faster/errors"
)

// MessageHandler is a function that returns a message.
type MessageHandler func() []byte

// PowHandler is an interface that defines the methods for the PoW handler.
type PowHandler interface {
	GenerateHash(msg []byte, nonce int) []byte
	IsValidHash(hash []byte, byteIndex int, byteValue byte) bool
	GetClientConditions(clientAddr net.Addr) (byteIndex int, byteValue byte)
}

type Dependencies struct {
	TCPAddress     string         `validate:"required"`
	MessageHandler MessageHandler `validate:"required"`
	PowHandler     PowHandler     `validate:"required"`
}

func (d *Dependencies) SetDefaults() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(d)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	listener   net.Listener
	msgHandler MessageHandler
	pow        PowHandler
}

func New(deps *Dependencies) (*Server, error) {
	deps.SetDefaults()

	listener, err := net.Listen("tcp", deps.TCPAddress)
	if err != nil {
		return nil, errors.Wrap(err, "get a listener")
	}

	tcp := &Server{
		listener:   listener,
		msgHandler: deps.MessageHandler,
		pow:        deps.PowHandler,
	}
	return tcp, nil
}

func (h *Server) Listen(ctx context.Context) {
	done := make(chan struct{})

	go func() {
		<-ctx.Done()
		h.listener.Close() // close the listener to stop accepting new connections
		close(done)        // signal that shutdown is complete
	}()

	for {
		select {
		case <-done:
			return
		default:
			conn, err := h.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}

			go h.handleTCPConnection(ctx, conn)
		}
	}
}

//nolint:gocognit // it's a server method
func (h *Server) handleTCPConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	nonce := rand.Intn(pow.PowDigestLength) - 1 //nolint:gosec // it's ok here
	primaryHash := h.pow.GenerateHash(nil, nonce)
	byteIndex, byteValue := h.pow.GetClientConditions(conn.RemoteAddr())
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msgSize int32
			if err := binary.Read(conn, binary.BigEndian, &msgSize); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				continue
			}

			if msgSize <= 0 {
				log.WithField("s", msgSize).Error("incorrect message size")
				continue
			}

			log.WithField("s", msgSize).Debug("read message size")

			msgBuffer := make([]byte, msgSize)
			n, err := conn.Read(msgBuffer)
			if err != nil {
				log.WithError(err).Error("read message")
				continue
			}

			var protoMessage powerV1.Message
			if err = proto.Unmarshal(msgBuffer[:n], &protoMessage); err != nil {
				log.WithError(err).Error("unmarshal message")
				continue
			}

			var body []byte

			command := protoMessage.GetCommand()

			//nolint:exhaustive // ok
			switch command {
			case powerV1.CommandType_Connect:
				body = common.ConvetVerfyMessageToBytes(primaryHash, byteIndex, byteValue)
				log.WithField("body", string(body)).Debug("a connect message")

			case powerV1.CommandType_Content:
				clientHash := h.pow.GenerateHash(primaryHash, common.GetNonceFromMessage(protoMessage.GetBody()))
				isValid := h.pow.IsValidHash(clientHash, byteIndex, byteValue)
				if !isValid {
					command = powerV1.CommandType_ErrInvalidHash
				} else {
					body = h.msgHandler()
				}

				log.WithFields(log.Fields{"is_valid": isValid, "nonce": nonce}).
					Debug("a content message")

			case powerV1.CommandType_Close:
				return
			}

			if len(body) > 0 || command > powerV1.CommandType_Content {
				msg := powerV1.Message{Command: command, Body: body}

				var response []byte
				response, err = proto.Marshal(&msg)
				if err != nil {
					log.WithError(err).Error("marshal message")
					continue
				}

				size := sizeOfMessage(response)
				if err = binary.Write(conn, binary.BigEndian, size); err != nil {
					log.WithError(err).Error("write message size")
					continue
				}

				log.WithFields(log.Fields{"size": size, "command": command}).
					Debug("send a message")

				if _, err = conn.Write(response); err != nil {
					log.WithError(err).Warn("write message")
				}
			}
		}
	}
}

func sizeOfMessage(msg []byte) int32 {
	return int32(len(msg))
}
