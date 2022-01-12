package config

import "fmt"

type Config struct {
	ClientHost     string   `envconfig:"CLIENT_HOST"`
	ClientPort     string   `envconfig:"CLIENT_PORT"`
	ServerHost     string   `envconfig:"SERVER_HOST"`
	ServerPort     string   `envconfig:"SERVER_PORT"`
	QuotesFileName string   `envconfig:"FILE_NAME"`
	TCPaddrs       []string `envconfig:"TCP_ADDRS"`
}

func (c *Config) ClientAddr() string {
	return fmt.Sprintf("%s:%s", c.ClientHost, c.ClientPort)
}

func (c *Config) ServerAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}
