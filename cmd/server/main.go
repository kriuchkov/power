package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"power/internal/config"
	"power/internal/protomsg"
	"power/internal/services/server"

	"github.com/apibillme/cache"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

const cacheTime = 2 * time.Second

func main() {
	cache := cache.New(128, cache.WithTTL(cacheTime))
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		oscall := <-c
		log.Printf("system call: %+v", oscall)
		cancel()
	}()

	powDebug, _ := strconv.ParseBool(os.Getenv("POW_DEBUG"))
	if powDebug {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading .env: %s", err.Error())
		}
		log.SetLevel(log.DebugLevel)
	}

	var conf config.Config
	err := envconfig.Process("server", &conf)
	if err != nil {
		log.Fatalf("Error loading config: %s", err.Error())
	}

	quotesRaw, err := ioutil.ReadFile(conf.QuotesFileName)
	if err != nil {
		log.Fatalf("couldn't open a file with quotes %s", err.Error())
	}

	quotes := bytes.Split(quotesRaw, []byte("\n"))
	if len(quotes) == 0 {
		log.Fatal("quotes == 0")
	}

	handler := func() *protomsg.Message {
		msg := &protomsg.Message{
			Command: protomsg.CommandTypeMsg,
			Body:    []byte(quotes[rand.Intn(len(quotes))]),
		}
		return msg
	}

	rand.Seed(time.Now().UnixNano())
	addrs := make([]*net.TCPAddr, 0, len(conf.TCPaddrs))
	for i := range conf.TCPaddrs {
		addrs = append(addrs, getResolveTCPAddr(conf.TCPaddrs[i]))
	}
	serv, err := server.NewUDP(conf.ServerHost, conf.ServerPort, cache, addrs, server.WithHandler(handler))
	if err != nil {
		log.Fatalf("couldn't create a new server instance %s", err.Error())
	}

	go serv.Listen(ctx)
	<-ctx.Done()
	log.Println("server exited properly")
}

func getResolveTCPAddr(address string) *net.TCPAddr {
	t, _ := net.ResolveTCPAddr("tcp", address)
	return t
}
