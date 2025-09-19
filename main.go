package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ArpitKhatri1/gather.town/game" // replace with the real module path
)

func main() {
	// Serve the static HTML file
	http.Handle("/", http.FileServer(http.Dir("./game"))) // serves player.html

	// WebSocket endpoint
	http.HandleFunc("/ws", game.HandlePlayerJoin)

	// Start game loop
	go game.GameLoop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	fmt.Println("Game server running on http://localhost:" + port)
	log.Fatal(srv.ListenAndServe())
}
