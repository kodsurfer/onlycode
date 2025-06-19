package config

import "time"

type Config struct {
	HttpPort     string
	DockerApiVer string
	Timeout      time.Duration
	MaxConn      int
}
