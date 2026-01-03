// ship-service/main.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Ship struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

var fleet = []Ship{
	{"Old Bessie", true},
	{"The Dinghy", true},
	{"Leela's Cruiser", true},
}
var (
	ship = Ship{
		Name:      "Planet Express Ship",
		Available: true,
	}
	mu sync.Mutex
)

func getStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request for ship status")
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(ship)
}

func reserveShip(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to reserve ship")
	mu.Lock()
	defer mu.Unlock()
	if ship.Available {
		http.Error(w, "Ship is already in use", http.StatusConflict)
		return
	}
	ship.Available = false
	json.NewEncoder(w).Encode(ship)
}

func returnShip(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to return ship")
	mu.Lock()
	defer mu.Unlock()
	ship.Available = true
	json.NewEncoder(w).Encode(ship)
}

func main() {
	http.HandleFunc("/ship/status", getStatus)
	http.HandleFunc("/ship/reserve", reserveShip)
	http.HandleFunc("/ship/return", returnShip)
	http.ListenAndServe(":8080", nil)
}
