package main

import (
	"apiChecker/internal/client"
	"apiChecker/internal/config"

	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config : %v", err)
	}

	catalogClient := client.NewCatalogClient(cfg.Catalog, cfg.Internal)
}
