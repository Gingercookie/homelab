// ship-service/main.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Ship struct {
	Name      string     `json:"name"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"lock"`
}

type ShipInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

var fleet = []Ship{
	{"Old Bessie", true, sync.Mutex{}},
	{"The Dinghy", true, sync.Mutex{}},
	{"Leela's Cruiser", true, sync.Mutex{}},
}

func getStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request for ship status")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ship := r.URL.Query().Get("ship")
	if ship == "" {
		http.Error(w, "Missing ship name in status request", http.StatusBadRequest)
		return
	}

	found := false
	for i := range fleet {
		if fleet[i].Lock.TryLock() {
			if fleet[i].Name == ship {
				found = true
				json.NewEncoder(w).Encode(ShipInfo{fleet[i].Name, fleet[i].Available})
			}

			fleet[i].Lock.Unlock()
			if found {
				return
			}
		}
	}

	http.NotFound(w, r)
}

func reserveShip(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to reserve ship")

	found := false
	for i := range fleet {
		if fleet[i].Lock.TryLock() {
			if fleet[i].Available {
				found = true
				fmt.Printf("[INFO] Ship %s is available\n", fleet[i].Name)
				fleet[i].Available = false
				fmt.Printf("[INFO] Ship %s has been reserved and is no longer available\n", fleet[i].Name)
				json.NewEncoder(w).Encode(ShipInfo{fleet[i].Name, fleet[i].Available})
			}

			fleet[i].Lock.Unlock()
			if found {
				return
			}
		}
	}

	fmt.Println("[WARN] No ship is available")
	http.Error(w, "No ship available", http.StatusServiceUnavailable)
}

func returnShip(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to return ship")

	var ship ShipInfo
	if err := json.NewDecoder(r.Body).Decode(&ship); err != nil {
		http.Error(w, "Failed to unmarshal data into ship member", http.StatusServiceUnavailable)
	}

	for i := range fleet {
		if fleet[i].Name == ship.Name {
			fmt.Printf("[INFO] Returning ship %s to base\n", fleet[i].Name)
			fleet[i].Lock.Lock()
			fleet[i].Available = true
			fleet[i].Lock.Unlock()
			fmt.Printf("[INFO] Ship %s has been returned and is now available\n", fleet[i].Name)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.NotFound(w, r)
}

func main() {
	http.HandleFunc("/ship/status", getStatus)
	http.HandleFunc("/ship/reserve", reserveShip)
	http.HandleFunc("/ship/return", returnShip)
	http.ListenAndServe(":8080", nil)
}
