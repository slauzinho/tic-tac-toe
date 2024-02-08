package main

import (
	"log"
	"net/http"

	"github.com/slauzinho/tic-tac-toe/api"
)

func main() {
	http.HandleFunc("/ws", api.PlayGame)
	log.Fatal(http.ListenAndServe(":4000", nil))
}
