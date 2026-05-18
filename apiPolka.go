package main

import (
	"net/http"
	"fmt"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/Dxnax-RS/Chirpy/internal/auth"
)

type newPolkaRequest struct{
	Event 	string `json:"event"`
	Data 	polkaData `json:"data"`
}

type polkaData struct{
	UserID uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) updateUserToRed() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		key, err := auth.GetAPIKey(req.Header)
		if err != nil {
			errorMesage := fmt.Sprintf("Server response: %s", err)
			respondWithError(w, 401, errorMesage)
			return
		}
		if key != cfg.polkaKey {
			errorMesage := fmt.Sprintf("Authorization missing",)
			respondWithError(w, 401, errorMesage)
			return
		}
		decoder := json.NewDecoder(req.Body)
		newRequest := newPolkaRequest{}
		err = decoder.Decode(&newRequest)
		if err != nil {
			errorMesage := fmt.Sprintf("Error handling JSON: %s", err)
			respondWithError(w, 500, errorMesage)
			return
		}

		response := NoResponse{}

		if newRequest.Event != "user.upgraded" {
			respondWithJSON(w, 204, response)
		}

		err = cfg.queries.UpdateToRed(context.Background(), newRequest.Data.UserID)
		if err != nil {
			errorMesage := fmt.Sprintf("DB responded with error: %s", err)
			respondWithError(w, 404, errorMesage)
			return
		}

		respondWithJSON(w, 204, response)
	}
}