package main

import (
	"log"
	"net/http"

	"github.com/PovilasDuo/Infocenter-service/internal/handler"
)

func main() {
	http.HandleFunc("/infocenter/", handler.InfoCenterHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
