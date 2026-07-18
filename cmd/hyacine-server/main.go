package main

import (
	"log"

	"hyacine-go-server/internal/config"
	"hyacine-go-server/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(httpapi.ListenAndServe(cfg))
}
