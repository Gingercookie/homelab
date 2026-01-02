// delivery-simulator/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type DeliveryRequest struct {
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Contents  string `json:"contents"`
}

var (
	apiURL = getEnv("API_URL", "http://planetexpress-api/deliveries")
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

var recipients = []string{
	"Philip J. Fry", "Turanga Leela", "Bender Bending Rodríguez",
	"Amy Wong", "Hermes Conrad", "Professor Farnsworth", "Zapp Brannigan", "Kif Kroker",
	"Mom", "Elzar", "Scruffy", "Robot Santa", "Calculon",
}

var addresses = []string{
	"New New York", "Mars Vegas", "Neptune", "Omicron Persei 8", "Robonia",
	"Luna Park", "Doop Headquarters", "Sewer City", "Central Bureaucracy",
}

var contents = []string{
	"Slurm", "Popplers", "Shiny metal parts", "Career chips", "Dark matter",
	"Mutant fish", "Love potion", "Explosives", "Robot oil", "Hyper-chicken eggs",
}

func randomChoice(list []string) string {
	return list[rand.Intn(len(list))]
}

func sendDelivery() {
	req := DeliveryRequest{
		Recipient: randomChoice(recipients),
		Address:   randomChoice(addresses),
		Contents:  randomChoice(contents),
	}

	data, _ := json.Marshal(req)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("[ERROR] Failed to send delivery: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("[INFO] Sent delivery: %+v | Status: %d\n", req, resp.StatusCode)
}

func main() {
	interval := 5 * time.Second
	if val := os.Getenv("INTERVAL_SECONDS"); val != "" {
		if secs, err := time.ParseDuration(val + "s"); err == nil {
			interval = secs
		}
	}

	fmt.Printf("DeliverySimulator running, sending requests to %s every %v\n", apiURL, interval)

	for {
		sendDelivery()
		time.Sleep(interval)
	}
}
