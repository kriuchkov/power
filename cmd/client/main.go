package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	"power/internal/config"
	"power/internal/pow"
	"power/internal/provider/power"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

func main() {
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

	conn, err := net.ListenUDP("udp", getResolveUDPAddr(conf.ClientAddr()))
	if err != nil {
		log.Fatalf("couldn't open a udp connection: %s", conf.ServerAddr())
	}
	defer conn.Close()

	ctx := context.Background()
	client := power.New(ctx, conn, getResolveUDPAddr(conf.ServerAddr()), pow.SolveHash)
	msg, err := client.GetMessage(ctx)
	if err != nil {
		log.Fatalf("couldn't read message from server %s", err.Error())
	}

	fmt.Println(string(msg))
	client.Close()
}

func getResolveUDPAddr(address string) *net.UDPAddr {
	t, _ := net.ResolveUDPAddr("udp", address)
	return t
}
