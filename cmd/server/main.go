package main

import (
	"log"

	"github.com/pedroj/go-inventory-management/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("server foundation configured for %s at %s", cfg.Database.Host, cfg.Server.Address)
}
