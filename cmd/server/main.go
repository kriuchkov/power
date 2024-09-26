package main

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"os/signal"
	"strconv"

	"github.com/kriuchkov/power/internal/config"
	"github.com/kriuchkov/power/internal/pow"
	"github.com/kriuchkov/power/pkg/server"

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
		log.WithError(err).Fatal("process the config")
	}

	log.WithField("config", conf).Info("config loaded")

	quotesRaw, err := os.ReadFile(conf.QuotesFileName)
	if err != nil {
		log.WithError(err).Fatal("read quotes file")
	}

	quotes := bytes.Split(quotesRaw, []byte("\n"))
	if len(quotes) == 0 {
		log.Panic("quotes file is empty")
	}

	serv, err := server.New(&server.Dependencies{
		TCPAddress:     conf.ServerAddr,
		PowHandler:     pow.NewPow(conf.Difficulty),
		MessageHandler: func() []byte { return quotes[rand.Intn(len(quotes))] }, //nolint:gosec // it's ok here
	})
	if err != nil {
		log.WithError(err).Fatal("create a new server")
	}

	log.WithField("address", conf.ServerAddr).Info("server started")
	go serv.Listen(ctx)

	<-ctx.Done()
	log.Println("server exited properly")
}
