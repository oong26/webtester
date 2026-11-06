package configs

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr string
	Database Database
	Zap      Zap
	Redis    Redis
}

type Database struct {
	Path string
}

type Zap struct {
	Image          string
	TimeoutSeconds int
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

func Load() Config {
	return Config{
		HTTPAddr: getenv("HTTP_ADDR", ":9090"),
		Database: Database{Path: getenv("DB_PATH", "webtester.db")},
		Zap: Zap{
			Image:          getenv("ZAP_IMAGE", "ghcr.io/zaproxy/zaproxy:stable"),
			TimeoutSeconds: 1800,
		},
		Redis: Redis{
			Addr:     getenv("REDIS_ADDR", "localhost:6379"),
			Password: getenv("REDIS_PASSWORD", ""),
			DB:       getenvInt("REDIS_DB", 0),
		},
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func getenvInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		var i int
		_, _ = fmt.Sscanf(v, "%d", &i)
		return i
	}
	return d
}
