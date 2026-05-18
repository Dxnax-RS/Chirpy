package main

import _ "github.com/lib/pq"

import (
	"net/http"
	"log"
	"fmt"
	"time"
	"sync/atomic"
	"encoding/json"
	"database/sql"
	"os"
	"github.com/joho/godotenv"
	"github.com/google/uuid"
	"github.com/Dxnax-RS/Chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits 	atomic.Int32
	queries 		*database.Queries
	env 			string
	jwtSecret 		string
	polkaKey		string
}

type User struct {
	ID 				uuid.UUID 	`json:"id"`
	CreatedAt 		time.Time 	`json:"created_at"`
	UpdatedAt 		time.Time 	`json:"updated_at"`
	Email 			string 		`json:"email"`
	Token 			string 		`json:"token"`
	RefreshToken 	string		`json:"refresh_token"`
	IsChirpyRed		bool		`json:"is_chirpy_red"`
}

type Chirp struct {
	ID 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Body 		string 		`json:"body"`
	UserID 		uuid.UUID 	`json:"user_id"`
}

type JWT struct{
	Token string `json:"token"`
}

type NoResponse struct{}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		_ = cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
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

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil{
		log.Fatalf("Error connecting to database: %v", err)
	}
	dbQueries := database.New(db)

	var apiCfg apiConfig
	apiCfg.queries = dbQueries
	apiCfg.env = os.Getenv("ENVIRONMENT")
	apiCfg.jwtSecret = os.Getenv("JWT_SECRET")
	apiCfg.polkaKey = os.Getenv("POLKA_KEY")
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
	mux.HandleFunc("GET /api/chirps", apiCfg.getAllChirps())
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirp())
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirp())
	mux.HandleFunc("POST /api/users", apiCfg.createUser())
	mux.HandleFunc("POST /api/login", apiCfg.login())
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshJWT())
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshToken())
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.updateUserToRed())
	mux.HandleFunc("PUT /api/users", apiCfg.updateUser())
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp())
	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics())
	mux.HandleFunc("POST /admin/reset", apiCfg.reset())

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}