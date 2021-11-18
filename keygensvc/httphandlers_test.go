package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockKgService struct {
	key string
	error error
}

func (m MockKgService) GetGeneratedKey(_ string, _ int) (string, error) {
	return m.key, m.error
}

func (m MockKgService) StoreCustomKey(_ string, _ string) error {
	return m.error
}
var OriginalKgService KeyGenService

func init() {
	OriginalKgService = App.Kg
}

func TestHandleGenerateKeyRequest(t *testing.T) {
	t.Run("returns 405 Method Not Allowed when disallowed method is detected", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest("GET", "/key/generate", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Received %d, expected %d", status, http.StatusMethodNotAllowed)
		}
		if res.Header().Get("Allow") != http.MethodPost {
			t.Errorf("Received %s, expected %s", res.Header().Get("Allow"), http.MethodPost)
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 422 Unprocessable Entity when request JSON cannot be parsed", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest("POST", "/key/generate", strings.NewReader("{]"))
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("Received %d, expected %d", status, http.StatusUnprocessableEntity)
		}
		if res.Body.String() != "Could not parse request JSON.\n" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Could not parse request JSON.")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 400 Bad Request when key length is invalid", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/generate",
			strings.NewReader(`{"source_name": "my-source", "key_length": 0}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String()[:21] != "Key length is invalid" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Key length is invalid")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 400 Bad Request when source name length is invalid", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/generate",
			strings.NewReader(`{"source_name": "src", "key_length": 8}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String()[:29] != "Source name length is invalid" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Source name length is invalid")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 500 Internal Server Error when key cannot be generated", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: errors.New("failed")}
		req, err := http.NewRequest(
			"POST",
			"/key/generate",
			strings.NewReader(`{"source_name": "my-source", "key_length": 8}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusInternalServerError {
			t.Errorf("Received %d, expected %d", status, http.StatusInternalServerError)
		}
		if res.Body.String() != "Internal server error: Failed to process request.\n" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Internal server error: Failed to process request.")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 201 Created when key generation is successful", func(t *testing.T) {
		App.Kg = MockKgService{key: "12345678", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/generate",
			strings.NewReader(`{"source_name": "my-source", "key_length": 8}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleGenerateKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusCreated {
			t.Errorf("Received %d, expected %d", status, http.StatusCreated)
		}
		if res.Body.String() == "" {
			t.Error("Received nothing, expected populated body")
		}
		App.Kg = OriginalKgService
	})
}

func TestHandleNewKeyRequest(t *testing.T) {
	t.Run("returns 405 Method Not Allowed when disallowed method is detected", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest("GET", "/key/new", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Received %d, expected %d", status, http.StatusMethodNotAllowed)
		}
		if res.Header().Get("Allow") != http.MethodPost {
			t.Errorf("Received %s, expected %s", res.Header().Get("Allow"), http.MethodPost)
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 422 Unprocessable Entity when request JSON cannot be parsed", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest("POST", "/key/new", strings.NewReader("{]"))
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("Received %d, expected %d", status, http.StatusUnprocessableEntity)
		}
		if res.Body.String() != "Could not parse request JSON.\n" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Could not parse request JSON.")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 400 Bad Request when key length is invalid", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/new",
			strings.NewReader(`{"key": "123"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String()[:21] != "Key length is invalid" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Key length is invalid")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 400 Bad Request when source name length is invalid", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/new",
			strings.NewReader(`{"source_name": "src", "key": "123"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String()[:21] != "Key length is invalid" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Key length is invalid")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 500 Internal Server Error when new key cannot be stored", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: errors.New("failed")}
		req, err := http.NewRequest(
			"POST",
			"/key/new",
			strings.NewReader(`{"key": "12345678"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusInternalServerError {
			t.Errorf("Received %d, expected %d", status, http.StatusInternalServerError)
		}
		if res.Body.String() != "Internal server error: Failed to process request.\n" {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Internal server error: Failed to process request.")
		}
		App.Kg = OriginalKgService
	})
	t.Run("returns 201 Created when new key storage is successful", func(t *testing.T) {
		App.Kg = MockKgService{key: "", error: nil}
		req, err := http.NewRequest(
			"POST",
			"/key/new",
			strings.NewReader(`{"key": "12345678"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		h := http.HandlerFunc(HandleNewKeyRequest)
		h.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusCreated {
			t.Errorf("Received %d, expected %d", status, http.StatusCreated)
		}
		if res.Body.String() != "" {
			t.Error("Received response body, expected nothing")
		}
		App.Kg = OriginalKgService
	})
}
