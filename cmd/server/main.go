package main

import (
	"log"
	"net/http"

	httpadapter "github.com/pedroj/go-inventory-management/internal/adapter/http"
	"github.com/pedroj/go-inventory-management/internal/adapter/postgres"
	"github.com/pedroj/go-inventory-management/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := postgres.Open(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repository, err := postgres.NewProductRepository(db)
	if err != nil {
		log.Fatal(err)
	}
	productHandler, err := httpadapter.NewProductHandler(repository)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/products/new", productHandler)
	mux.Handle("/products", productHandler)
	mux.HandleFunc("/health", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})

	log.Printf("starting HTTP server at %s", cfg.Server.Address)
	if err := http.ListenAndServe(cfg.Server.Address, mux); err != nil {
		log.Fatal(err)
	}
}
