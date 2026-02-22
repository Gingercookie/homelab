// package-service/main.go
package main

import (
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"
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

func listPackages(w http.ResponseWriter, _ *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	list := make([]Package, 0, len(packages))
	for _, pkg := range packages {
		list = append(list, pkg)
	}
	json.NewEncoder(w).Encode(list)
}

func getPackage(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request to get a package")
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
	slog.Info("Received request to update package status")
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
		slog.Info("Successfully updated package status", "id", pkg.ID, "status", status)
	} else {
		http.NotFound(w, r)
		slog.Warn("Package was not found", "id", id)
	}
}

func deletePackage(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received request to delete package")
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing package id in delete request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	if _, ok := packages[id]; !ok {
		http.NotFound(w, r)
		return
	}

	delete(packages, id)
	w.WriteHeader(http.StatusOK)
}

func randomID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
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

	http.HandleFunc("/packages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createPackage(w, r)
		} else {
			listPackages(w, r)
		}
	})
	http.HandleFunc("/packages/get", getPackage)
	http.HandleFunc("/packages/update", updatePackageStatus)
	http.HandleFunc("/packages/delete", deletePackage)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
