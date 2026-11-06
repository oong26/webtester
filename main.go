package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"time"

	"webtester/internal/server"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address (keep it on localhost)")
	flag.Parse()

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(sub),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("webtester UI on http://%s", *addr)
	log.Fatal(srv.ListenAndServe())
}
