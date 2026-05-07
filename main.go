package main

import _ "github.com/lib/pq"

import (
	"net/http"
	"log"
	"fmt"
	"time"
	"strings"
	"context"
	"sync/atomic"
	"encoding/json"
	"database/sql"
	"os"
	"github.com/joho/godotenv"
	"github.com/google/uuid"
	"github.com/Dxnax-RS/Chirpy/internal/database"
	"github.com/Dxnax-RS/Chirpy/internal/auth"
)

type apiConfig struct {
	fileserverHits 	atomic.Int32
	queries 		*database.Queries
	env 			string
	jwtSecret 		string
}

type User struct {
	ID 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Email 		string 		`json:"email"`
	Token 		string 		`json:"token"`
}

type Chirp struct {
	ID 			uuid.UUID 	`json:"id"`
	CreatedAt 	time.Time 	`json:"created_at"`
	UpdatedAt 	time.Time 	`json:"updated_at"`
	Body 		string 		`json:"body"`
	UserID 		uuid.UUID 	`json:"user_id"`
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

func (cfg *apiConfig) getAllChirps() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		dbresponse, err := cfg.queries.GetAllChirps(context.Background())
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		var respBody []Chirp

		for _, v := range dbresponse {
			res := Chirp{
				ID: v.ID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
				Body: v.Body,
				UserID: v.UserID,
			}
			respBody = append(respBody, res)
		}

		respondWithJSON(w, 200, respBody)
	}
}

func (cfg *apiConfig) getChirp() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		stringUUID := req.PathValue("chirpID")
		chirpUUID, err := uuid.Parse(stringUUID)
		if err != nil {
			errorMesage := fmt.Sprintf("UUID parsing issue: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		dbresponse, err := cfg.queries.GetChirp(context.Background(), chirpUUID)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 404, errorMesage)
			return
		}

		respBody := Chirp{
			ID: dbresponse.ID,
			CreatedAt: dbresponse.CreatedAt,
			UpdatedAt: dbresponse.UpdatedAt,
			Body: dbresponse.Body,
			UserID: dbresponse.UserID,
		}

		respondWithJSON(w, 200, respBody)
	}
}

func (cfg *apiConfig) login() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type newUserRequest struct{
			Password 			string 	`json:"password"`
			Email 				string 	`json:"email"`
			ExpiresInSeconds 	int 	`json:"expires_in_seconds"`
		}
		
		decoder := json.NewDecoder(req.Body)
		newRequest := newUserRequest{}
		err := decoder.Decode(&newRequest)

		if err != nil {
			errorMesage := fmt.Sprintf("Error handling JSON: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		dbresponse, err := cfg.queries.GetUser(context.Background(), newRequest.Email)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 404, errorMesage)
			return
		}

		respBody := User{
			ID: dbresponse.ID,
			CreatedAt: dbresponse.CreatedAt,
			UpdatedAt: dbresponse.UpdatedAt,
			Email: dbresponse.Email,
		}

		match, err := auth.CheckPasswordHash(newRequest.Password, dbresponse.HashedPassword)

		if err != nil {
			errorMesage := fmt.Sprintf("Error hashing pasword: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		if !match{
			errorMesage := fmt.Sprintf("Incorrect email or password")
			respondWithError(w, 401, errorMesage)
			return
		}

		var expiresIn time.Duration

		if newRequest.ExpiresInSeconds == 0 || newRequest.ExpiresInSeconds > 3600{
			expiresIn = 3600 * time.Second
		}else{
			expiresIn = time.Duration(newRequest.ExpiresInSeconds) * time.Second
		}

		tokenString, err := auth.MakeJWT(respBody.ID, cfg.jwtSecret, expiresIn)

		if err != nil {
			errorMesage := fmt.Sprintf("Error generating token: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		respBody.Token = tokenString

		respondWithJSON(w, 200, respBody)
	}
}

func (cfg *apiConfig) createChirp() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type newChirpRequest struct{
			Body 	string 		`json:"body"`
			UserID 	uuid.UUID 	`json:"user_id"`
		}

		tokenString, err := auth.GetBearerToken(req.Header)
		if err != nil {
			errorMesage := fmt.Sprintf("Error geting bearer token: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		userID, err := auth.ValidateJWT(tokenString, cfg.jwtSecret)
		if err != nil {
			errorMesage := fmt.Sprintf("Error validating token: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		rep := strings.NewReplacer("kerfuffle", "****", "sharbert", "****", "fornax", "****", "Kerfuffle", "****", "Sharbert", "****", "Fornax", "****")

		decoder := json.NewDecoder(req.Body)
		myChirp := newChirpRequest {}
		err = decoder.Decode(&myChirp)
		if err != nil {
			respondWithError(w, 400, "Error handling JSON")
			return
		}

		myChirp.UserID = userID

		if len(myChirp.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		myChirp.Body = rep.Replace(myChirp.Body)

		params := database.CreateChirpParams{
			Body: myChirp.Body,
			UserID: myChirp.UserID,
		}

		dbresponse, err := cfg.queries.CreateChirp(context.Background(), params)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		respBody := Chirp{
			ID: dbresponse.ID,
			CreatedAt: dbresponse.CreatedAt,
			UpdatedAt: dbresponse.UpdatedAt,
			Body: dbresponse.Body,
			UserID: dbresponse.UserID,
		}
		respondWithJSON(w, 201, respBody)
	}
}

func (cfg *apiConfig) createUser() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type newUserRequest struct{
			Password 	string `json:"password"`
			Email 		string `json:"email"`
		}

		decoder := json.NewDecoder(req.Body)
		newRequest := newUserRequest{}
		err := decoder.Decode(&newRequest)
		if err != nil {
			errorMesage := fmt.Sprintf("Error handling JSON: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		hash, err := auth.HashPassword(newRequest.Password)
		if err != nil {
			errorMesage := fmt.Sprintf("Hashing error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		params := database.CreateUserParams{
			Email: newRequest.Email,
			HashedPassword: hash,
		}

		dbresponse, err := cfg.queries.CreateUser(context.Background(), params)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		respBody := User{
			ID: dbresponse.ID,
			CreatedAt: dbresponse.CreatedAt,
			UpdatedAt: dbresponse.UpdatedAt,
			Email: dbresponse.Email,
		}
		respondWithJSON(w, 201, respBody)
	}
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
	mux.HandleFunc("GET /admin/metrics", apiCfg.metrics())
	mux.HandleFunc("POST /admin/reset", apiCfg.reset())

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
}