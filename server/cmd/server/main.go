// Screen Timer server entrypoint.
//
// Starts an HTTP server that serves the REST API (for the management UI and
// the Windows agent) and static frontend assets from ./static.
// The listen port defaults to 8080 and can be overridden with the PORT
// environment variable.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/octachrome/screen-timer/server/internal/server"
)

func main() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "data/screen-timer.json"
	}
	store := server.NewStoreWithFile(dataFile)
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
