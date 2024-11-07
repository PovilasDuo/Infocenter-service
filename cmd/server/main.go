package main

import (
	"log"
	"net/http"

	"github.com/PovilasDuo/Infocenter-service/internal/handler"
	"github.com/PovilasDuo/Infocenter-service/pubsub"
)

func main() {
	ps := pubsub.NewPubSub()

	mux := http.NewServeMux()
	mux.Handle("/infocenter/", http.StripPrefix("/infocenter", handler.NewInfoCenterHandler(ps)))

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
