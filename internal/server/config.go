package server

import (
	"time"
)

type Config struct {
	Host                   string        `yaml:"host"`
	Port                   int           `yaml:"port"`
	TLSCertFile            string        `yaml:"tlsCertFile"`
	TLSKeyFile             string        `yaml:"tlsKeyFile"`
	TLSReloadInterval      time.Duration `yaml:"tlsReloadInterval"`
	HTTPSRedirect          bool          `yaml:"httpsRedirect"`
	AntiCheatBuckets       int           `yaml:"antiCheatBuckets"`
	AntiCheatPeriod        time.Duration `yaml:"antiCheatPeriod"`
	AntiCheatMaxConcurrent int           `yaml:"antiCheatMaxConcurrent"`
	DataDir                string        `yaml:"dataDir"`
	ShutdownTimeout        time.Duration `yaml:"shutdownTimeout"`
	AdminKey               string        `yaml:"adminKey"` // this is very secure, don't worry
	PuzzleOrder            []string      `yaml:"puzzleOrder"`
	Deadline               time.Time     `yaml:"deadline"`
}
