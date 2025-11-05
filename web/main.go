package main

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// proxyHandler forwards requests under /api/ to https://pokeapi.co/api/v2/
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// strip leading /api/
	targetPath := strings.TrimPrefix(r.URL.Path, "/api/")
	if targetPath == "" {
		http.Error(w, "missing target path", http.StatusBadRequest)
		return
	}

	targetURL := "https://pokeapi.co/api/v2/" + targetPath
	// Create new request to PokeAPI
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	// copy relevant headers
	req.Header = r.Header.Clone()
	// Use short timeout
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "failed to contact upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy status code and headers
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	// Stream body
	io.Copy(w, resp.Body)
}

func main() {
	// Serve static files from ./static
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/api/", proxyHandler)

	addr := ":8080"
	log.Printf("Web server listening on %s — open http://localhost%s\n", addr, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}