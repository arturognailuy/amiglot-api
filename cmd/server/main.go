package main

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gnailuy/amiglot-api/internal/config"
	"github.com/gnailuy/amiglot-api/internal/db"
	httpserver "github.com/gnailuy/amiglot-api/internal/http"
)

func main() {
	cfg := config.Load()

	pool, err := db.New(cfg)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if pool != nil {
		defer pool.Close()
		log.Printf("database connected")
	} else {
		log.Printf("DATABASE_URL not set; starting without database")
	}

	if err := runServer(cfg, pool, http.ListenAndServe); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

type listenFunc func(addr string, handler http.Handler) error

func runServer(cfg config.Config, pool *pgxpool.Pool, listen listenFunc) error {
	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	return listen(addr, httpserver.Router(cfg, pool))
}
