// ship-service/main.go
package main

import (
	"encoding/json"
	"fmt"
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
	fmt.Println("[INFO] Received request for ship status")
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(ship)
}

func reserveShip(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to reserve ship")
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
	fmt.Println("[INFO] Received request to return ship")
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
