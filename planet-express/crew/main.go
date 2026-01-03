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
	var available []CrewMember
	for i := range crew {
		if crew[i].Lock.TryLock() {
			if crew[i].Available {
				fmt.Printf("[INFO] Crew member %s is available\n", crew[i].Name)
				crew[i].Available = false
				json.NewEncoder(w).Encode(CrewResponse{crew[i].Name})
			}
			crew[i].Lock.Unlock()
		}
	}

	if len(available) == 0 {
		fmt.Println("[WARN] No crew is available")
	}
}

func returnCrew(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to return a crew member")

	var c CrewMember
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Failed to unmarshal data into crew member", 503)
	}

	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.Available = true

	fmt.Printf("Crew member %s was returned successfully\n", c.Name)
	json.NewEncoder(w).Encode(CrewResponse{c.Name})
}

func main() {
	http.HandleFunc("/crew/reserve", reserveCrew)
	http.HandleFunc("/crew/return", returnCrew)
	http.ListenAndServe(":8080", nil)
}
