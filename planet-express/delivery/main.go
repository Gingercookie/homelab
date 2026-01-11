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
	Name string `json:"name"`
}

type ShipStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
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

func requestAvailableCrew() (CrewMember, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/crew/reserve", crewServiceURL))
	resp, err := http.Get(fmt.Sprintf("%s/crew/reserve", crewServiceURL))
	if err != nil {
		return CrewMember{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from crew service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	fmt.Printf("[DEBUG] Parsed body of response: %s", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return CrewMember{}, resp.StatusCode, fmt.Errorf("crew service error: %s", bodyBytes)
	}
	fmt.Printf("[DEBUG] Status code is OKAY\n")

	var crew CrewMember
	if err := json.Unmarshal(bodyBytes, &crew); err != nil {
		return CrewMember{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %s\n", crew)

	return crew, http.StatusOK, nil
}

func reserveShip() (ShipStatus, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/ship/reserve", shipServiceURL))
	resp, err := http.Post(fmt.Sprintf("%s/ship/reserve", shipServiceURL), "application/json", nil)
	if err != nil {
		return ShipStatus{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from ship service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ShipStatus{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}
	fmt.Printf("[DEBUG] Parsed body of response: %s", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return ShipStatus{}, resp.StatusCode, fmt.Errorf("ship reservation failed: %s", string(bodyBytes))
	}

	fmt.Printf("[DEBUG] Status code is OKAY\n")
	var ship ShipStatus
	if err := json.Unmarshal(bodyBytes, &ship); err != nil {
		return ShipStatus{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %v\n", ship)

	return ship, resp.StatusCode, nil
}

func createPackage(pkg Package) (Package, int, error) {
	fmt.Printf("[DEBUG] Sending request to %s\n", fmt.Sprintf("%s/packages", packageServiceURL))

	data, _ := json.Marshal(pkg)
	resp, err := http.Post(fmt.Sprintf("%s/packages", packageServiceURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return Package{}, http.StatusServiceUnavailable, err
	}
	defer resp.Body.Close()

	fmt.Printf("[DEBUG] Got response from package service\n")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Package{}, resp.StatusCode, fmt.Errorf("error reading response body")
	}

	if resp.StatusCode != http.StatusCreated {
		return Package{}, resp.StatusCode, fmt.Errorf("package creation failed: %s", string(bodyBytes))
	}

	fmt.Printf("[DEBUG] Status code is OKAY\n")
	var created Package
	if err := json.Unmarshal(bodyBytes, &created); err != nil {
		return Package{}, resp.StatusCode, err
	}
	fmt.Printf("[DEBUG] Unmarshaled into %s\n", created)

	return created, resp.StatusCode, nil
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("[INFO] Got request for new delivery")
	var req DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("[INFO] Dispatching request for available crew")
	crew, statusCode, err := requestAvailableCrew()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}
	fmt.Printf("[DEBUG] Got crew member %s\n", crew.Name)

	fmt.Println("[INFO] Dispatching request to reserve ship")
	ship, statusCode, err := reserveShip()
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	fmt.Println("[INFO] Got both crew member and ship")
	if (crew == CrewMember{}) || (ship == ShipStatus{}) {
		http.Error(w, "Unable to get ship or crew", http.StatusServiceUnavailable)
		return
	}

	fmt.Println("[INFO] Dispatching request to create new package")
	pkg, statusCode, err := createPackage(Package{
		Recipient: req.Recipient,
		Address:   req.Address,
		Contents:  req.Contents,
	})
	if err != nil {
		http.Error(w, err.Error(), statusCode)
		return
	}

	// Build the delivery ticket
	ticket := DeliveryTicket{
		Crew:    crew,
		Ship:    ship,
		Package: pkg,
	}
	fmt.Printf("[INFO] Delivery ticket created: %v\n", ticket)

	// Simulate random delivery time
	go func(pkgID string, crew CrewMember) {
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
		fmt.Println("[INFO] Returning ship to base")
		resp, err = http.Post(fmt.Sprintf("%s/ship/return", shipServiceURL), "application/json", nil)
		if err != nil {
			fmt.Println("[ERROR] Failed to return ship:", err)
		} else {
			resp.Body.Close()
			fmt.Println("[INFO] Ship returned to base.")
		}

		// Return crew to base
		fmt.Printf("[INFO] Returning crew member %s to base", crew.Name)
		data, _ := json.Marshal(crew)
		resp, err = http.Post(fmt.Sprintf("%s/crew/return", crewServiceURL), "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("[ERROR] Failed to return crew member %s to base:%s", crew.Name, err)
		} else {
			resp.Body.Close()
			fmt.Printf("[INFO] Crew member %s returned to base.", crew.Name)
		}
	}(pkg.ID, crew)

	// Send ticket to requester
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ticket)
}

func main() {
	http.HandleFunc("/deliveries", handleDelivery)
	fmt.Println("DeliveryService running on :8080")
	http.ListenAndServe(":8080", nil)
}
