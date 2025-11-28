// ship-service/main.go
package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

type ShipStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "available" or "in-flight"
}

var (
	ship = ShipStatus{
		Name:   "Planet Express Ship",
		Status: "available",
	}
	mu sync.Mutex
)

func getStatus(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(ship)
}

func reserveShip(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	if ship.Status == "in-flight" {
		http.Error(w, "Ship is already in use", http.StatusConflict)
		return
	}
	ship.Status = "in-flight"
	json.NewEncoder(w).Encode(ship)
}

func returnShip(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	ship.Status = "available"
	json.NewEncoder(w).Encode(ship)
}

func main() {
	http.HandleFunc("/ship/status", getStatus)
	http.HandleFunc("/ship/reserve", reserveShip)
	http.HandleFunc("/ship/return", returnShip)
	http.ListenAndServe(":8080", nil)
}
