package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type generateKeyRequestJson struct {
	SourceName string `json:"source_name"`
	KeyLength  int    `json:"key_length"`
}

type generateKeyResponseJson struct {
	Key string `json:"key"`
}

func HandleGenerateKeyRequest(w http.ResponseWriter, r *http.Request) {
	log.Print("/key/generate hit")

	// Check method for validity
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("%s /key/generate allowed", r.Method)

	// Parse request
	var bodyBuff bytes.Buffer
	bodyBuff.ReadFrom(r.Body)
	var requestJson generateKeyRequestJson
	jsonUnmarshalErr := json.Unmarshal([]byte(bodyBuff.String()), &requestJson)
	if jsonUnmarshalErr != nil {
		log.Printf(
			"Error parsing the generate key request JSON: %s", jsonUnmarshalErr,
		)
		http.Error(
			w, "Could not parse request JSON.", http.StatusUnprocessableEntity,
		)
		return
	}
	log.Print("Request JSON parsed")

	// Validate request
	if requestJson.KeyLength < App.EnvVars.MinKeyLength ||
				requestJson.KeyLength > App.EnvVars.MaxKeyLength {
		log.Print("Key length invalid")
		http.Error(
			w,
			fmt.Sprintf(
				"Key length is invalid, must be >%d and <%d",
				App.EnvVars.MinKeyLength,
				App.EnvVars.MaxKeyLength,
			),
			http.StatusBadRequest,
		)
		return
	}
	if len(requestJson.SourceName) < App.EnvVars.MinSourceNameLength {
		log.Print("Source name length invalid")
		http.Error(
			w,
			fmt.Sprintf(
				"Source name length is invalid, must be >%d",
				App.EnvVars.MinSourceNameLength,
			),
			http.StatusBadRequest,
		)
		return
	}
	log.Print("Request JSON validated")

	// Get generated key
	log.Print("Getting generated key...")
	key, err := App.Kg.GetGeneratedKey(requestJson.SourceName, requestJson.KeyLength)
	if err != nil {
		log.Printf("Error getting generated key: %s", err)
		http.Error(
			w,
			"Internal server error: Failed to process request.",
			http.StatusInternalServerError,
		)
		return
	}
	log.Print("Key generated")

	// Encode response JSON
	encodedJson, _ := json.Marshal(generateKeyResponseJson{Key: key})
	log.Print("Response encoded")

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(encodedJson)
}

type newKeyRequestJson struct {
	SourceName string `json:"source_name"`
	Key        string `json:"key"`
}

func HandleNewKeyRequest(w http.ResponseWriter, r *http.Request) {
	log.Print("/key/new hit")

	// Check method for validity
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var bodyBuff bytes.Buffer
	bodyBuff.ReadFrom(r.Body)
	var requestJson newKeyRequestJson
	jsonUnmarshalErr := json.Unmarshal([]byte(bodyBuff.String()), &requestJson)
	if jsonUnmarshalErr != nil {
		log.Printf(
			"Error parsing the new key request JSON: %s", jsonUnmarshalErr,
		)
		http.Error(
			w, "Could not parse request JSON.", http.StatusUnprocessableEntity,
		)
		return
	}

	// Validate request
	if len(requestJson.Key) < App.EnvVars.MinKeyLength ||
				len(requestJson.Key) > App.EnvVars.MaxKeyLength {
		log.Print("Key length invalid")
		http.Error(
			w,
			fmt.Sprintf(
				"Key length is invalid, must be >%d and <%d",
				App.EnvVars.MinKeyLength,
				App.EnvVars.MaxKeyLength,
			),
			http.StatusBadRequest,
		)
		return
	}
	if requestJson.SourceName != "" &&
				len(requestJson.SourceName) < App.EnvVars.MinSourceNameLength {
		log.Print("Source name length invalid")
		http.Error(
			w,
			fmt.Sprintf(
				"Source name length is invalid, must be >%d",
				App.EnvVars.MinSourceNameLength,
			),
			http.StatusBadRequest,
		)
		return
	}
	log.Print("Request JSON validated")

	// Store custom key
	err := App.Kg.StoreCustomKey(requestJson.SourceName, requestJson.Key)
	if err != nil {
		log.Printf("Error storing custom key: %s", err)
		http.Error(
			w,
			"Internal server error: Failed to process request.",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
