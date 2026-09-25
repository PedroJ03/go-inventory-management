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
	locationRepository, err := postgres.NewLocationRepository(db)
	if err != nil {
		log.Fatal(err)
	}
	stockLevelRepository, err := postgres.NewStockLevelRepository(db)
	if err != nil {
		log.Fatal(err)
	}
	locationHandler, err := httpadapter.NewLocationHandler(locationRepository, stockLevelRepository)
	if err != nil {
		log.Fatal(err)
	}
	receivingRepository, err := postgres.NewStockReceivingRepository(db)
	if err != nil {
		log.Fatal(err)
	}
	receiveStockHandler, err := httpadapter.NewReceiveStockHandler(repository, locationRepository, receivingRepository)
	if err != nil {
		log.Fatal(err)
	}
	sellingRepository, err := postgres.NewStockSellingRepository(db)
	if err != nil {
		log.Fatal(err)
	}
	sellStockHandler, err := httpadapter.NewSellStockHandler(repository, locationRepository, sellingRepository)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/products/new", productHandler)
	mux.Handle("/products", productHandler)
	mux.Handle("/locations/new", locationHandler)
	mux.Handle("/locations", locationHandler)
	mux.Handle("/locations/", locationHandler)
	mux.Handle("/stock/receive", receiveStockHandler)
	mux.Handle("/stock/sell", sellStockHandler)
	mux.HandleFunc("/health", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})

	log.Printf("starting HTTP server at %s", cfg.Server.Address)
	if err := http.ListenAndServe(cfg.Server.Address, mux); err != nil {
		log.Fatal(err)
	}
}
