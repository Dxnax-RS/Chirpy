package main

import (
	"net/http"
	"fmt"
	"context"
)

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
		if cfg.env != "dev" {
			errorMesage := fmt.Sprintf("Forbidden action")
			respondWithError(w, 403, errorMesage)
			return
		}
		
		err := cfg.queries.ResetUsers(context.Background())

		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		data := []byte("OK")
		cfg.fileserverHits.Store(0)
		_, err = w.Write(data)
		if err != nil {
			fmt.Println("Error sending response")
		}
	}
}