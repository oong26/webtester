package configs

import "os"

type Config struct {
	HTTPAddr string
	Database Database
	Zap      Zap
}

type Database struct {
	Path string
}

type Zap struct {
	Image          string
	TimeoutSeconds int
}

func Load() Config {
	return Config{
		HTTPAddr: getenv("HTTP_ADDR", ":9090"),
		Database: Database{Path: getenv("DB_PATH", "webtester.db")},
		Zap: Zap{
			Image:          getenv("ZAP_IMAGE", "ghcr.io/zaproxy/zaproxy:stable"),
			TimeoutSeconds: 1800,
		},
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
