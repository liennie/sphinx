package server

import (
	"time"
)

type Config struct {
	Port                   int           `yaml:"port"`
	AntiCheatBuckets       int           `yaml:"antiCheatBuckets"`
	AntiCheatPeriod        time.Duration `yaml:"antiCheatPeriod"`
	AntiCheatMaxConcurrent int           `yaml:"antiCheatMaxConcurrent"`
	DataDir                string        `yaml:"dataDir"`
	ShutdownTimeout        time.Duration `yaml:"shutdownTimeout"`
	AdminKey               string        `yaml:"adminKey"` // this is very secure, don't worry
}
