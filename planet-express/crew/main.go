// crew-service/main.go
package main

import (
	"encoding/json"
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
	var available []CrewMember
	for _, member := range crew {
		if member.Status == "available" {
			available = append(available, member)
		}
	}
	json.NewEncoder(w).Encode(available)
}

func main() {
	http.HandleFunc("/crew/available", getAvailableCrew)
	http.ListenAndServe(":8080", nil)
}
