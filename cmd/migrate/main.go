package main

import (
	"log"

	"github.com/pedroj/go-inventory-management/internal/config"
	"github.com/pedroj/go-inventory-management/internal/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if err := migrations.Run(cfg, "."); err != nil {
		log.Fatal(err)
	}
	log.Println("database migrations applied successfully")
}
