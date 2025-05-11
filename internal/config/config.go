package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env     string        `yaml:"env" env-default:"dev"`
	Timeout time.Duration `yaml:"timeout"`
	GRPC    GRPCConfig    `yaml:"grpc"`
}

type GRPCConfig struct {
	Port int `yaml:"port"`
}

func MustLoad() *Config {
	path := fetchConfigFlag()
	if path == "" {
		panic("path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}
	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic("config reading error: " + err.Error())
	}
	return &cfg
}

func fetchConfigFlag() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	return res
}
