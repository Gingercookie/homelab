// crew-service/main.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type CrewMember struct {
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Available bool       `json:"available"`
	Lock      sync.Mutex `json:"lock"`
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
	fmt.Println("[INFO] Received request to reserve a crew member")
	found := false
	for i := range crew {
		if crew[i].Lock.TryLock() {
			if crew[i].Available {
				found = true
				fmt.Printf("[INFO] Crew member %s is available\n", crew[i].Name)
				crew[i].Available = false
				fmt.Printf("[INFO] Crew member %s is no longer available\n", crew[i].Name)
				json.NewEncoder(w).Encode(CrewResponse{crew[i].Name})
			}
			crew[i].Lock.Unlock()
			// Need to unlock the mutex before we return
			if found {
				return
			}
		}
	}

	fmt.Println("[WARN] No crew is available")
	http.Error(w, "No crew available", http.StatusServiceUnavailable)
}

func returnCrew(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to return a crew member")

	var c CrewMember
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Failed to unmarshal data into crew member", 503)
	}

	for i := range crew {
		if crew[i].Name == c.Name {
			crew[i].Lock.Lock()
			crew[i].Available = true
			crew[i].Lock.Unlock()

			fmt.Printf("[INFO] Crew member %s was returned successfully\n", c.Name)
			json.NewEncoder(w).Encode(CrewResponse{c.Name})
			return
		}
	}

	http.Error(w, "Crew member not found", http.StatusNotFound)
}

func main() {
	http.HandleFunc("/crew/reserve", reserveCrew)
	http.HandleFunc("/crew/return", returnCrew)
	http.ListenAndServe(":8080", nil)
}
