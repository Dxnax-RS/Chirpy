package main

import (
	"net/http"
	"fmt"
	"time"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/Dxnax-RS/Chirpy/internal/database"
	"github.com/Dxnax-RS/Chirpy/internal/auth"
)

func (cfg *apiConfig) login() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		type newUserRequest struct{
			Password 			string 	`json:"password"`
			Email 				string 	`json:"email"`
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
			IsChirpyRed: dbresponse.IsChirpyRed,
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

		expiresIn := 3600 * time.Second

		tokenString, err := auth.MakeJWT(respBody.ID, cfg.jwtSecret, expiresIn)

		if err != nil {
			errorMesage := fmt.Sprintf("Error generating token: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		respBody.Token = tokenString

		refreshToken := auth.MakeRefreshToken()

		params := database.CreateRefreshTokenParams{
			Token: refreshToken,
			UserID: respBody.ID,
			ExpiresAt: time.Now().Add(1440 * time.Hour),
		}

		tokendbresponse, err := cfg.queries.CreateRefreshToken(context.Background(), params)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		respBody.RefreshToken = tokendbresponse.Token

		respondWithJSON(w, 200, respBody)
	}
}

func (cfg *apiConfig) refreshJWT() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		tokenString, err := auth.GetBearerToken(req.Header)
		if err != nil {
			errorMesage := fmt.Sprintf("Error geting bearer token: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		dbresponse , err := cfg.queries.GetUserFromRefreshToken(context.Background(), tokenString)
		if err != nil || dbresponse == uuid.Nil{
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		expiresIn := 1 * time.Hour

		responseTokenString, err := auth.MakeJWT(dbresponse, cfg.jwtSecret, expiresIn)

		response := JWT{
			Token: responseTokenString,
		}

		respondWithJSON(w, 200, response)
	}
}

func (cfg *apiConfig) revokeRefreshToken() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		tokenString, err := auth.GetBearerToken(req.Header)
		if err != nil {
			errorMesage := fmt.Sprintf("Error geting bearer token: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		err = cfg.queries.RevokeRefreshToken(context.Background(), tokenString)
		if err != nil{
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}

		response := NoResponse{}

		respondWithJSON(w, 204, response)
	}
}