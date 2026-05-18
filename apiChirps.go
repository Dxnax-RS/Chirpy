package main

import (
	"net/http"
	"fmt"
	"strings"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/Dxnax-RS/Chirpy/internal/database"
	"github.com/Dxnax-RS/Chirpy/internal/auth"
)

func (cfg *apiConfig) getAllChirps() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		dbresponse, err := cfg.queries.GetAllChirps(context.Background())
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		var response []Chirp

		for _, v := range dbresponse {
			res := Chirp{
				ID: v.ID,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
				Body: v.Body,
				UserID: v.UserID,
			}
			response = append(response, res)
		}

		respondWithJSON(w, 200, response)
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

		response := Chirp{
			ID: dbresponse.ID,
			CreatedAt: dbresponse.CreatedAt,
			UpdatedAt: dbresponse.UpdatedAt,
			Body: dbresponse.Body,
			UserID: dbresponse.UserID,
		}

		respondWithJSON(w, 200, response)
	}
}

func (cfg *apiConfig) deleteChirp() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		stringUUID := req.PathValue("chirpID")
		chirpUUID, err := uuid.Parse(stringUUID)

		if err != nil {
			errorMesage := fmt.Sprintf("UUID parsing issue: %s", err)
			respondWithError(w, 500, errorMesage)
			return
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

		requestedChirp, err := cfg.queries.GetChirp(context.Background(), chirpUUID)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 404, errorMesage)
			return
		}

		if requestedChirp.UserID != userID{
			errorMesage := fmt.Sprintf("Forbiden action")
			respondWithError(w, 403, errorMesage)
			return
		}

		err = cfg.queries.DeleteChirp(context.Background(), requestedChirp.ID)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 404, errorMesage)
			return
		}

		response := NoResponse{}

		respondWithJSON(w, 204, response)
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

		requestedChirp, err := cfg.queries.CreateChirp(context.Background(), params)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		response := Chirp{
			ID: requestedChirp.ID,
			CreatedAt: requestedChirp.CreatedAt,
			UpdatedAt: requestedChirp.UpdatedAt,
			Body: requestedChirp.Body,
			UserID: requestedChirp.UserID,
		}
		respondWithJSON(w, 201, response)
	}
}