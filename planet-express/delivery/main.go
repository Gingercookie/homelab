// delivery-service/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"math/rand/v2"
)

type CrewMember struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type ShipStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Package struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Status    string `json:"status"`
	Contents  string `json:"contents"`
}

type DeliveryRequest struct {
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Contents  string `json:"contents"`
}

type DeliveryTicket struct {
	Crew    CrewMember `json:"crew"`
	Ship    ShipStatus `json:"ship"`
	Package Package    `json:"package"`
}

var (
	crewServiceURL    = getEnv("CREW_SERVICE_URL", "http://crew-service")
	shipServiceURL    = getEnv("SHIP_SERVICE_URL", "http://ship-service")
	packageServiceURL = getEnv("PACKAGE_SERVICE_URL", "http://package-service")
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func requestAvailableCrew() (CrewMember, error) {
	resp, err := http.Get(fmt.Sprintf("%s/crew/available", crewServiceURL))
	if err != nil {
		return CrewMember{}, err
	}
	defer resp.Body.Close()

	var crew []CrewMember
	if err := json.NewDecoder(resp.Body).Decode(&crew); err != nil {
		return CrewMember{}, err
	}
	if len(crew) == 0 {
		return CrewMember{}, fmt.Errorf("no crew available")
	}
	return crew[0], nil
}

func reserveShip() (ShipStatus, error) {
	resp, err := http.Post(fmt.Sprintf("%s/ship/reserve", shipServiceURL), "application/json", nil)
	if err != nil {
		return ShipStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ShipStatus{}, fmt.Errorf("ship reservation failed: %s", string(body))
	}
	var ship ShipStatus
	if err := json.NewDecoder(resp.Body).Decode(&ship); err != nil {
		return ShipStatus{}, err
	}
	return ship, nil
}

func createPackage(pkg Package) (Package, error) {
	data, _ := json.Marshal(pkg)
	resp, err := http.Post(fmt.Sprintf("%s/packages", packageServiceURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return Package{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return Package{}, fmt.Errorf("package creation failed: %s", string(body))
	}
	var created Package
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return Package{}, err
	}
	return created, nil
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	crew, err := requestAvailableCrew()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	ship, err := reserveShip()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	pkg, err := createPackage(Package{
		Recipient: req.Recipient,
		Address:   req.Address,
		Contents:  req.Contents,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Build the delivery ticket
	ticket := DeliveryTicket{
		Crew:    crew,
		Ship:    ship,
		Package: pkg,
	}

	// Simulate random delivery time
	go func(pkgID string) {
		delay := time.Duration(rand.IntN(5)+1) * time.Second
		fmt.Printf("[INFO] Ship in-flight for %v delivering package %s\n", delay, pkgID)
		time.Sleep(delay)

		// Mark package as delivered
		resp, err := http.Get(fmt.Sprintf("%s/packages/update?id=%s&status=delivered", packageServiceURL, pkgID))
		if err != nil {
			fmt.Println("[ERROR] Failed to update package status:", err)
		} else {
			resp.Body.Close()
			fmt.Printf("[INFO] Package %s marked as delivered\n", pkgID)
		}

		// Return ship to base
		resp, err = http.Post(fmt.Sprintf("%s/ship/return", shipServiceURL), "application/json", nil)
		if err != nil {
			fmt.Println("[ERROR] Failed to return ship:", err)
		} else {
			resp.Body.Close()
			fmt.Println("[INFO] Ship returned to base.")
		}
	}(pkg.ID)

	// Send ticket to requester
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func main() {
	http.HandleFunc("/deliveries", handleDelivery)
	fmt.Println("DeliveryService running on :8080")
	http.ListenAndServe(":8080", nil)
}
