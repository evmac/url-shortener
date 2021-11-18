package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type MockKgsClient struct {
	response *http.Response
	error error
}

func (m MockKgsClient) PostJson(_ string, _ json.RawMessage) (*http.Response, error) {
	return m.response, m.error
}

func TestKgsService_GenerateKey(t *testing.T) {
	t.Run("returns error when KGS API Generate Key call fails", func(t *testing.T) {
		mockKgsClient := MockKgsClient{response: nil, error: errors.New("failed")}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		_, genErr := kgsSvc.GenerateKey("some-source", 12)
		if genErr != ErrKgsCouldNotProcessRequest {
			t.Errorf("Received %s, expected %s", genErr, ErrKgsCouldNotProcessRequest)
		}
	})
	t.Run("returns error when status code is not 201 Created", func(t *testing.T) {
		mockKgsClient := MockKgsClient{
			response: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader("")),
			}, error: nil,
		}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		_, genErr := kgsSvc.GenerateKey("some-source", 12)
		if genErr != ErrKgsCouldNotFulfillRequest {
			t.Errorf("Received %s, expected %s", genErr, ErrKgsCouldNotFulfillRequest)
		}
	})
	t.Run("returns error when response JSON cannot be parsed", func(t *testing.T) {
		mockKgsClient := MockKgsClient{
			response: &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(strings.NewReader(`{]`)),
			},
			error: nil,
		}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		_, genErr := kgsSvc.GenerateKey("some-source", 12)
		if genErr != ErrCouldNotParseResponseJson {
			t.Errorf("Received %s, expected %s", genErr, ErrCouldNotParseResponseJson)
		}
	})
	t.Run("returns key when successful", func(t *testing.T) {
		mockKgsClient := MockKgsClient{
			response: &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(strings.NewReader(`{"key": "12345"}`)),
			},
			error: nil,
		}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		key, genErr := kgsSvc.GenerateKey("some-source", 12)
		if genErr != nil {
			t.Errorf("Received %s, expected nil", genErr)
		}
		if key != "12345" {
			t.Errorf("Received %s, expected %s", key, "12345")
		}
	})
}

func TestKgsService_CreateNewKey(t *testing.T) {
	t.Run("returns error when KGS API Create New Key call fails", func(t *testing.T) {
		mockKgsClient := MockKgsClient{response: nil, error: errors.New("failed")}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		_, genErr := kgsSvc.CreateNewKey("some-source", "12345")
		if genErr != ErrKgsCouldNotProcessRequest {
			t.Errorf("Received %s, expected %s", genErr, ErrKgsCouldNotProcessRequest)
		}
	})
	t.Run("returns error when status code is not 201 Created", func(t *testing.T) {
		mockKgsClient := MockKgsClient{
			response: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader("")),
			}, error: nil,
		}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		_, genErr := kgsSvc.CreateNewKey("some-source", "12345")
		if genErr != ErrKgsCouldNotFulfillRequest {
			t.Errorf("Received %s, expected %s", genErr, ErrKgsCouldNotFulfillRequest)
		}
	})
	t.Run("returns key when successful", func(t *testing.T) {
		mockKgsClient := MockKgsClient{
			response: &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(strings.NewReader("")),
			},
			error: nil,
		}
		kgsSvc, _ := NewKgsService(mockKgsClient)
		key, genErr := kgsSvc.CreateNewKey("some-source", "12345")
		if genErr != nil {
			t.Errorf("Received %s, expected nil", genErr)
		}
		if key != "12345" {
			t.Errorf("Received %s, expected %s", key, "12345")
		}
	})
}