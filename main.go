package main

import (
	"net/http"
	"log"
	"fmt"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		_ = cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		hits := cfg.fileserverHits.Load()
		msg := fmt.Sprintf("Hits: %d", hits)
		data := []byte(msg)
		_, err := w.Write(data)
		if err != nil {
			fmt.Println("Error sending response")
		}
	}
}

func (cfg *apiConfig) reset() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		data := []byte("OK")
		cfg.fileserverHits.Store(0)
		_, err := w.Write(data)
		if err != nil {
			fmt.Println("Error sending response")
		}
	}
}

func main() {
	var apiCfg apiConfig
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	var srv http.Server
	srv.Addr = ":8080"
	srv.Handler = mux

	healthz := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		data := []byte("OK")
		_, err := w.Write(data)
		if err != nil {
			fmt.Println("Error sending response")
		}
	}

	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/metrics", apiCfg.metrics())
	mux.HandleFunc("/reset", apiCfg.reset())

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}