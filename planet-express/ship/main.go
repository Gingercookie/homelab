// ship-service/main.go
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

type Ship struct {
	Name      string     `json:"name"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"-"`
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
	slog.Info("Received request for ship status")
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
	slog.Info("Received request to reserve ship")

	found := false
	for i := range fleet {
		if fleet[i].Lock.TryLock() {
			if fleet[i].Available {
				found = true
				slog.Info("Ship is available", "name", fleet[i].Name)
				fleet[i].Available = false
				slog.Info("Ship has been reserved", "name", fleet[i].Name)
				json.NewEncoder(w).Encode(ShipInfo{fleet[i].Name, fleet[i].Available})
			}

			fleet[i].Lock.Unlock()
			if found {
				return
			}
		}
	}

	slog.Warn("No ship is available")
	http.Error(w, "No ship available", http.StatusServiceUnavailable)
}

func returnShip(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request to return ship")

	var ship ShipInfo
	if err := json.NewDecoder(r.Body).Decode(&ship); err != nil {
		http.Error(w, "Failed to unmarshal data into ship member", http.StatusServiceUnavailable)
		return
	}

	for i := range fleet {
		if fleet[i].Name == ship.Name {
			slog.Info("Returning ship to base", "name", fleet[i].Name)
			fleet[i].Lock.Lock()
			fleet[i].Available = true
			fleet[i].Lock.Unlock()
			slog.Info("Ship returned and is now available", "name", fleet[i].Name)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.NotFound(w, r)
}

func main() {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = "INFO"
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	http.HandleFunc("/ship/status", getStatus)
	http.HandleFunc("/ship/reserve", reserveShip)
	http.HandleFunc("/ship/return", returnShip)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
