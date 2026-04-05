package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-permit/internal/server"
	"github.com/stockyard-dev/stockyard-permit/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9811"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./permit-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("permit: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Permit — Self-hosted permit and license tracking\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n", port, port)
	log.Printf("permit: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
