// crew-service/main.go
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

type CrewMember struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"-"`
}

type CrewResponse struct {
	Name string `json:"name"`
}

var crew = []CrewMember{
	{"Fry", "Delivery Boy", true, sync.Mutex{}},
	{"Leela", "Captain", true, sync.Mutex{}},
	{"Bender", "Bending Unit", true, sync.Mutex{}},
}

func reserveCrew(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request to reserve a crew member")
	found := false
	for i := range crew {
		if crew[i].Lock.TryLock() {
			if crew[i].Available {
				found = true
				slog.Info("Crew member is available", "name", crew[i].Name)
				crew[i].Available = false
				slog.Info("Crew member has been reserved", "name", crew[i].Name)
				json.NewEncoder(w).Encode(CrewResponse{crew[i].Name})
			}
			// Need to unlock the mutex before we return
			crew[i].Lock.Unlock()
			if found {
				return
			}
		}
	}

	slog.Warn("No crew is available")
	http.Error(w, "No crew available", http.StatusServiceUnavailable)
}

func returnCrew(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request to return a crew member")

	var c CrewMember
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Failed to unmarshal data into crew member", http.StatusServiceUnavailable)
		return
	}

	for i := range crew {
		if crew[i].Name == c.Name {
			crew[i].Lock.Lock()
			crew[i].Available = true
			crew[i].Lock.Unlock()

			slog.Info("Crew member returned successfully", "name", c.Name)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	http.Error(w, "Crew member not found", http.StatusNotFound)
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

	http.HandleFunc("/crew/reserve", reserveCrew)
	http.HandleFunc("/crew/return", returnCrew)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
