package config

type Config struct {
	ServerAddr     string `envconfig:"SERVER_ADDR" default:":9090"`
	Difficulty     int    `envconfig:"DIFFICULTY" default:"4"`
	QuotesFileName string `envconfig:"FILE_NAME"`
}
