package config

import (
	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

const (
	EnvFile         = ".env"
	EnvConfigPrefix = "STREAMDAL_LOG_PROCESSOR"
)

type Config struct {
	ListenAddr           string `kong:"default='0.0.0.0:6000',help='The address to listen on for TCP requests.'"`
	LogstashAddr         string `kong:"default='logstash-server:7002',help='The address of the logstash server to forward logs to.'"`
	StreamdalServer      string `kong:"default='localhost:8082',help='The address of the streamdal server to pull rules from'"`
	StreamdalToken       string `kong:"default='1234',help='The token to use to authenticate with the streamdal server'"`
	StreamdalServiceName string `kong:"default='logstash',help='The name of the service to use when registering with streamdal'"`

	KongContext *kong.Context `kong:"-"`
}

func New() *Config {
	if err := godotenv.Load(EnvFile); err != nil {
		log.Debug("unable to load dotenv file", "err", err.Error(), "filename", EnvFile)
	}

	cfg := &Config{}
	cfg.KongContext = kong.Parse(cfg,
		kong.Name("streamdal"),
		kong.Description("Streamdal CLI"),
		kong.DefaultEnvars(EnvConfigPrefix),
	)

	return cfg
}
