package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/kriuchkov/power/internal/config"
	"github.com/kriuchkov/power/internal/pow"
	"github.com/kriuchkov/power/pkg/client"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	powDebug, _ := strconv.ParseBool(os.Getenv("POW_DEBUG"))
	if powDebug {
		godotenv.Load(".env") //nolint:errcheck // ok for this case
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	}

	var conf config.Config
	err := envconfig.Process("server", &conf)
	if err != nil {
		log.Panicf("loading config: %s", err.Error())
	}

	log.WithField("config", conf).Info("config loaded")

	serverConn, err := net.Dial("tcp", conf.ServerAddr)
	if err != nil {
		log.Panicf("connect to server: %s", err.Error())
	}
	defer serverConn.Close()

	client := client.New(&client.Dependencies{ServerConn: serverConn, Hasher: pow.NewPow(conf.Difficulty)})
	if err != nil {
		log.WithError(err).Panic("create a new client")
	}

	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msg, err := client.GetMessage(ctx)
	if err != nil {
		log.WithError(err).Panic("get message")
	}

	log.WithField("message", string(msg)).Info("message received")
}
