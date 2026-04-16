package main

import (
	"net/http"
	"log"
	"fmt"
	"strings"
	"sync/atomic"
	"encoding/json"
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		hits := cfg.fileserverHits.Load()
		msg := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, hits)
		fmt.Fprint(w, msg)
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

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type ChirpError struct {
		Err string `json:"error"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	respBody := ChirpError{
		Err: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Write(dat)
}

func (cfg *apiConfig) validateChirp() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type chirp struct{
			Message string `json:"body"`
		}

		type returnVals struct {
			Cleaned string `json:"cleaned_body"`
		}

		rep := strings.NewReplacer("kerfuffle", "****", "sharbert", "****", "fornax", "****", "Kerfuffle", "****", "Sharbert", "****", "Fornax", "****")

		decoder := json.NewDecoder(req.Body)
		myChirp := chirp {}
		err := decoder.Decode(&myChirp)
		if err != nil {
			respondWithError(w, 400, "Error handling JSON")
			return
		}

		if len(myChirp.Message) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		myChirp.Message = rep.Replace(myChirp.Message)

		respBody := returnVals{
			Cleaned: myChirp.Message,
		}
		respondWithJSON(w, 200, respBody)
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

	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateChirp())
	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics())
	mux.HandleFunc("POST /admin/reset", apiCfg.reset())

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}