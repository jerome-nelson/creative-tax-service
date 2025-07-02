package main

import (
	"JiraConnect/shared"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type TokenPayload struct {
	GrantType    string `json:"grantType"`
	RedirectURI  string `json:"redirectUri"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Code         string `json:"code"`
	RefreshToken string `json:"refreshToken"`
}

func handleGenerateToken(log *log.Logger, config shared.JiraConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.Header.Get("X-Code")
		if code == "" {
			http.Error(w, "Missing X-Code", http.StatusBadRequest)
		}
		values := TokenPayload{
			GrantType:    "authorization_code",
			Code:         code,
			ClientID:     config.Cid,
			ClientSecret: config.Secret,
			RedirectURI:  config.RedirectUrl,
		}

		jsonValue, _ := json.Marshal(values)
		oauth, err := http.Post(
			config.OauthUrl,
			"application/json",
			bytes.NewBuffer(jsonValue),
		)

		if oauth != nil {
			if oauth.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(oauth.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := shared.Oauth{}
			err := json.NewDecoder(oauth.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("oauth error:", err)
				return
			}

			if res.AccessToken == "" {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth - JSON is empty")
			}

			shared.SetJiraCookie(w, log, res)
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}

		defer oauth.Body.Close()
	}
}

func handleRefreshToken(log *log.Logger, config shared.JiraConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		if token = r.Header.Get("x-refresh"); token == "" {
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
		}

		values := TokenPayload{
			GrantType:    "refresh_token",
			RefreshToken: token,
			ClientID:     config.Cid,
			ClientSecret: config.Secret,
		}

		jsonValue, _ := json.Marshal(values)
		oauth, err := http.Post(
			config.OauthUrl,
			"application/json",
			bytes.NewBuffer(jsonValue),
		)

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("Error closing body:", err)
			}
		}(oauth.Body)

		if oauth != nil {
			if oauth.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(oauth.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := shared.Oauth{}
			err := json.NewDecoder(oauth.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("failed to retrieve refresh: ", err)
				return
			}

			if res.AccessToken == "" {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("failed to retrieve refresh - JSON is empty")
			}

			shared.SetJiraCookie(w, log, res)
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}
	}

}

func handleTempIssue(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var anyJson map[string]interface{}
		jsonFile, err := os.ReadFile("./_bin/issues-1-sample.json")
		err2 := json.Unmarshal(jsonFile, &anyJson)
		if err != nil || err2 != nil {
			if err != nil {
				fmt.Println(err)
			}
			if err2 != nil {
				fmt.Println(err2)
			}
			http.Error(w, "Error retrieving file", http.StatusInternalServerError)
			return
		}

		log.Println("Successfully Opened ./jira/issues-1-sample.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(anyJson)
	}
}
