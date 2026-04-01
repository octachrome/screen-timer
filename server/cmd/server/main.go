package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/octachrome/screen-timer/server/internal/server"
)

func main() {
	store := server.NewStore()
	router := server.NewRouter(store)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Screen Timer server starting on %s", addr)
	if err := http.ListenAndServe(addr, server.EnableCORS(router)); err != nil {
		log.Fatal(err)
	}
}
