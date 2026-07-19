package main

import (
	"log"

	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/httpapi"
)

var version = "dev"

func main() {
	log.Printf("Hyacine Server %s", version)
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(httpapi.ListenAndServe(cfg))
}
