package main

// Generic module for REST interaction with Key Generation Service

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type KgsClient interface {
	PostJson(endpoint string, rawJson json.RawMessage) (*http.Response, error)
}

type kgsClient struct {
	kgsUrl string
}

func NewKgsClient(kgsUrl string) KgsClient {
	return &kgsClient{kgsUrl: kgsUrl}
}

func (c kgsClient) PostJson(endpoint string, rawJson json.RawMessage) (*http.Response, error) {
	// Make request
	return http.Post(
		c.kgsUrl + endpoint,
		"application/json",
		bytes.NewBuffer(rawJson),
	)
}

type KgsService interface {
	GenerateKey(sourceName string, keyLength int) (string, error)
	CreateNewKey(sourceName string, key string) (string, error)
}

type kgsService struct {
	Client KgsClient
}

func NewKgsService(kgsClient KgsClient) (KgsService, error) {
	return &kgsService{
		Client: kgsClient,
	}, nil
}

var (
	ErrKgsCouldNotProcessRequest = errors.New("keygensvc could not process request")
	ErrKgsCouldNotFulfillRequest = errors.New("keygensvc could not fulfill request")
	ErrCouldNotParseResponseJson = errors.New("could not parse response json")
)

type generateKeyRequestJson struct {
	SourceName string `json:"source_name"`
	KeyLength  int    `json:"key_length"`
}

type generateKeyResponseJson struct {
	Key string `json:"key"`
}

func (s kgsService) GenerateKey(sourceName string, keyLength int) (string, error) {
	// Construct payload
	requestJson, _ := json.Marshal(
		generateKeyRequestJson{SourceName: sourceName, KeyLength: keyLength},
	)

	// Make generate key request
	httpResponse, httpErr := s.Client.PostJson("/key/generate", requestJson)
	if httpErr != nil {
		log.Printf("Error posting /key/generate: %s", httpErr)
		return "", ErrKgsCouldNotProcessRequest
	}
	defer httpResponse.Body.Close()

	// Check status code
	if httpResponse.StatusCode != http.StatusCreated {
		log.Printf("[%d] Key was not generated", httpResponse.StatusCode)
		return "", ErrKgsCouldNotFulfillRequest
	}

	// Parse response
	rawJson := parseRawJsonFromHttpBody(httpResponse.Body)
	var responseJson generateKeyResponseJson
	parseErr := json.Unmarshal(rawJson, &responseJson)
	if parseErr != nil {
		log.Printf("Error parsing the response body for key generation: %s", parseErr)
		return "", ErrCouldNotParseResponseJson
	}

	// Return generated key
	log.Printf("[%d] Key generated: %s", httpResponse.StatusCode, responseJson.Key)
	return responseJson.Key, nil
}

type newKeyRequestJson struct {
	SourceName string `json:"source_name"`
	Key	   string `json:"key"`
}

func (s kgsService) CreateNewKey(sourceName string, key string) (string, error) {
	// Construct payload
	requestJson, _ := json.Marshal(
		newKeyRequestJson{SourceName: sourceName, Key: key},
	)

	// Make new key request
	httpResponse, httpErr := s.Client.PostJson("/key/new", requestJson)
	if httpErr != nil {
		log.Printf("Error posting /key/generate: %s", httpErr)
		return "", ErrKgsCouldNotProcessRequest
	}
	defer httpResponse.Body.Close()

	// Check status code
	if httpResponse.StatusCode != http.StatusCreated {
		log.Printf("[%d] Key was not created", httpResponse.StatusCode)
		return "", ErrKgsCouldNotFulfillRequest
	}

	// Return new key
	log.Printf("[%d] Key created: %s", httpResponse.StatusCode, key)
	return key, nil
}
