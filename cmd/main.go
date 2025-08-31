package main

import (
	"log"

	"http-mock-server/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	application := app.New()
	return application.Run()
}
