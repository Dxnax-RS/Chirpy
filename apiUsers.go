package main

import (
	"net/http"
	"fmt"
	"context"
	"encoding/json"
	"github.com/Dxnax-RS/Chirpy/internal/database"
	"github.com/Dxnax-RS/Chirpy/internal/auth"
)

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
			IsChirpyRed: dbresponse.IsChirpyRed,
		}
		respondWithJSON(w, 201, respBody)
	}
}

func (cfg *apiConfig) updateUser() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type newUserRequest struct{
			Password 	string `json:"password"`
			Email 		string `json:"email"`
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

		decoder := json.NewDecoder(req.Body)
		newRequest := newUserRequest{}
		err = decoder.Decode(&newRequest)
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

		params := database.UpdateUserParams{
			Email: newRequest.Email,
			HashedPassword: hash,
			ID: userID,
		}

		dbresponse, err := cfg.queries.UpdateUser(context.Background(), params)
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
			IsChirpyRed: dbresponse.IsChirpyRed,
		}
		respondWithJSON(w, 200, respBody)
	}
}