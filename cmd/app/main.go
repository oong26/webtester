package main

import (
	"log"
	"net/http"
	"time"

	"webtester/internal/auth"
	"webtester/internal/clients/zap"
	"webtester/internal/configs"
	"webtester/internal/routes"
	"webtester/internal/services"
	"webtester/internal/stores"
)

func main() {
	cfg := configs.Load()

	store, err := stores.NewSQLiteStore(cfg.Database.Path)
	if err != nil {
		log.Fatalf("scan store: %v", err)
	}
	loadStore, err := stores.NewLoadTestStore(cfg.Database.Path)
	if err != nil {
		log.Fatalf("load test store: %v", err)
	}

	userStore, err := auth.NewUserStore(cfg.Database.Path)
	if err != nil {
		log.Fatalf("user store: %v", err)
	}

	redisStore, err := auth.NewRedisStore(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("redis store: %v", err)
	}
	defer redisStore.Close()

	tokenConfig := auth.DefaultTokenConfig()
	authSvc := services.NewAuthService(userStore, redisStore, tokenConfig)
	authMiddleware := auth.NewMiddleware(redisStore, tokenConfig)

	zapClient := zap.NewDockerClient(
		cfg.Zap.Image,
		time.Duration(cfg.Zap.TimeoutSeconds)*time.Second,
	)
	scanSvc := services.NewScanService(store, zapClient)
	loadSvc := services.NewLoadTestService(loadStore)

	r := routes.New(cfg, scanSvc, loadSvc, authSvc, authMiddleware)
	log.Printf("webtester listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatal(err)
	}
}