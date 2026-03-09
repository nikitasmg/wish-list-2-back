package main

import (
	"log"

	"main/config"
	"main/internal/app"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	app.Run(cfg)
}
