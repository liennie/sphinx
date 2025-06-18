package server

import (
	"time"
)

type Config struct {
	Port                 int           `yaml:"port"`
	AntidosBuckets       int           `yaml:"antidosBuckets"`
	AntidosPeriod        time.Duration `yaml:"antidosPeriod"`
	AntidosMaxConcurrent int           `yaml:"antidosMaxConcurrent"`
	DataDir              string        `yaml:"dataDir"`
	ShutdownTimeout      time.Duration `yaml:"shutdownTimeout"`
}
