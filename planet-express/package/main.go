// package-service/main.go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Package struct {
	ID        string `json:"id"`
	Recipient string `json:"recipient"`
	Address   string `json:"address"`
	Status    string `json:"status"` // "pending", "in-transit", "delivered"
	Contents  string `json:"contents"`
}

var (
	packages = make(map[string]Package)
	mu       sync.Mutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func createPackage(w http.ResponseWriter, r *http.Request) {
	var pkg Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	pkg.ID = randomID()
	pkg.Status = "pending"
	packages[pkg.ID] = pkg
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pkg)
}

func listPackages(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	list := make([]Package, 0, len(packages))
	for _, pkg := range packages {
		list = append(list, pkg)
	}
	json.NewEncoder(w).Encode(list)
}

func getPackage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to get a package")
	id := r.URL.Query().Get("id")
	mu.Lock()
	defer mu.Unlock()
	if pkg, ok := packages[id]; ok {
		json.NewEncoder(w).Encode(pkg)
	} else {
		http.NotFound(w, r)
	}
}

func updatePackageStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[INFO] Received request to update package status")
	id := r.URL.Query().Get("id")
	status := r.URL.Query().Get("status")
	if status == "" {
		http.Error(w, "Missing status", http.StatusBadRequest)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if pkg, ok := packages[id]; ok {
		pkg.Status = status
		packages[id] = pkg
		json.NewEncoder(w).Encode(pkg)
		fmt.Printf("[INFO] Successfully updated status of package %s\n", pkg)
	} else {
		http.NotFound(w, r)
		fmt.Println("[WARN] Package was not found!")
	}
}

func randomID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func main() {
	http.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createPackage(w, r)
		} else {
			listPackages(w, r)
		}
	})
	http.HandleFunc("/packages/get", getPackage)
	http.HandleFunc("/packages/update", updatePackageStatus)
	http.ListenAndServe(":8080", nil)
}
