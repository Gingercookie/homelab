// crew-service/main.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type CrewMember struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"` // "available", "busy"
}

var crew = []CrewMember{
	{"Fry", "Delivery Boy", "available"},
	{"Leela", "Captain", "available"},
	{"Bender", "Bending Unit", "available"},
}

func getAvailableCrew(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request for crew")
	var available []CrewMember
	for _, member := range crew {
		if member.Status == "available" {
			fmt.Printf("[INFO] Crew member %s is available\n", member.Name)
			available = append(available, member)
		}
	}

	if len(available) == 0 {
		fmt.Println("[WARN] No crew is available")
	}

	json.NewEncoder(w).Encode(available)
}

func main() {
	http.HandleFunc("/crew/available", getAvailableCrew)
	http.ListenAndServe(":8080", nil)
}
