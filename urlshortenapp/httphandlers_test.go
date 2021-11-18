package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockUsService struct {
	esIsLive bool
	error error
	shortUrl string
	originalUrl string
}

func (m MockUsService) TestElasticsearchConnection() bool {
	return m.esIsLive
}

func (m MockUsService) RefreshElasticsearchIndex() error {
	return m.error
}

func (m MockUsService) ConstructShortUrlAndAssignToOriginalUrl(_ string, _ string, _ string, _ int) (string, error) {
	return m.shortUrl, m.error
}

func (_ MockUsService) constructShortUrl(_ string, _ string, _ int) (string, error) {
	return "", nil
}

func (_ MockUsService) assignShortUrlToOriginalUrl(_ string, _ string) error {
	return nil
}

func (m MockUsService) GetOriginalUrlForShortUrl(_ string) (string, error) {
	return m.originalUrl, m.error
}

var OriginalUsService UrlShortenService

func init() {
	OriginalUsService = App.UsService
}

func TestHandleIndexRequest(t *testing.T) {
	t.Run("returns 200 OK and an informative response", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusOK {
			t.Errorf("Received %d, expected %d", status, http.StatusOK)
		}
		if res.Body.String() != ResDefaultMessage {
			t.Errorf("Received %s, expected %s", res.Body.String(), ResDefaultMessage)
		}
		App.UsService = OriginalUsService
	})
}

func TestHandleHealthcheckRequest(t *testing.T) {
	t.Run("returns 503 Service Unavailable when health cannot be verified", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: false, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/healthcheck", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusServiceUnavailable {
			t.Errorf("Received %d, expected %d", status, http.StatusServiceUnavailable)
		}
		if res.Body.String() != fmt.Sprintf("Service unavailable: %s.", ResApplicationUnhealthy) {
			t.Errorf("Received %s, expected %s", res.Body.String(), fmt.Sprintf("Service unavailable: %s.", ResApplicationUnhealthy))
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 200 OK when health is verified", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/healthcheck", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusOK {
			t.Errorf("Received %d, expected %d", status, http.StatusOK)
		}
		if res.Body.String() != ResApplicationHealthy {
			t.Errorf("Received %s, expected %s", res.Body.String(), ResApplicationHealthy)
		}
		App.UsService = OriginalUsService
	})
}

func TestHandleUrlShortenRequest(t *testing.T) {
	t.Run("returns 405 Method Not Allowed when disallowed method is detected", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/url/shorten", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Received %d, expected %d", status, http.StatusMethodNotAllowed)
		}
		if res.Body.String() != ResMethodNotAllowed {
			t.Errorf("Received %s, expected %s", res.Body.String(), ResMethodNotAllowed)
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 422 Unprocessable Entity when request JSON cannot be parsed", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("POST", "/url/shorten", strings.NewReader("{]"))
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("Received %d, expected %d", status, http.StatusUnprocessableEntity)
		}
		if res.Body.String() != fmt.Sprintf("Unprocessable entity: %s.", ResCouldNotParseRequestJson) {
			t.Errorf("Received %s, expected %s", res.Body.String(), fmt.Sprintf("Unprocessable entity: %s.", ResCouldNotParseRequestJson))
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 400 Bad Request when validation errors occur", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest(
			"POST",
			"/url/shorten",
			strings.NewReader(
				`
				{
					"original_url": "definitely-fails",
					"short_url_host": "same-here",
					"custom_slug": "short",
					"slug_length": 1
				}`,
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String() == "" {
			t.Errorf("Received %s, expected populated body", res.Body.String())
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 500 Internal Server Error when short url cannot be constructed and assigned to original url", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: errors.New("failed"), shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest(
			"POST",
			"/url/shorten",
			strings.NewReader(
				`
				{
					"original_url": "http://successful.url/over/here?params=true",
					"short_url_host": "",
					"custom_slug": "",
					"slug_length": 8
				}`,
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusInternalServerError {
			t.Errorf("Received %d, expected %d", status, http.StatusInternalServerError)
		}
		if res.Body.String() != "Internal server error: Could not shorten URL http://successful.url/over/here?params=true." {
			t.Errorf("Received %s, expected %s", res.Body.String(), "Internal server error: Could not shorten URL http://successful.url/over/here?params=true.")
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 201 Created when successful", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "http://shrt.url/12345678", originalUrl: ""}
		req, err := http.NewRequest(
			"POST",
			"/url/shorten",
			strings.NewReader(
				`
				{
					"original_url": "http://successful.url/over/here?params=true",
					"short_url_host": "",
					"custom_slug": "",
					"slug_length": 8
				}`,
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusCreated {
			t.Errorf("Received %d, expected %d", status, http.StatusCreated)
		}
		if res.Body.String() == "" {
			t.Errorf("Received %s, expected populated body", res.Body.String())
		}
		App.UsService = OriginalUsService
	})
}

func TestHandleExternalUrlRedirect(t *testing.T) {
	t.Run("returns 405 Method Not Allowed when disallowed method is detected", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/url/redirect", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Received %d, expected %d", status, http.StatusMethodNotAllowed)
		}
		if res.Body.String() != ResMethodNotAllowed {
			t.Errorf("Received %s, expected %s", res.Body.String(), ResMethodNotAllowed)
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 422 Unprocessable Entity when request JSON cannot be parsed", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("POST", "/url/redirect", strings.NewReader("{]"))
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("Received %d, expected %d", status, http.StatusUnprocessableEntity)
		}
		if res.Body.String() != fmt.Sprintf("Unprocessable entity: %s.", ResCouldNotParseRequestJson) {
			t.Errorf("Received %s, expected %s", res.Body.String(), fmt.Sprintf("Unprocessable entity: %s.", ResCouldNotParseRequestJson))
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 400 Bad Request when validation errors occur", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest(
			"POST",
			"/url/redirect",
			strings.NewReader(`{"short_url": "definitely-fails"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusBadRequest {
			t.Errorf("Received %d, expected %d", status, http.StatusBadRequest)
		}
		if res.Body.String() == "" {
			t.Errorf("Received %s, expected populated body", res.Body.String())
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 500 Internal Server Error when original url cannot be retrieved for short url", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: errors.New("failed"), shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest(
			"POST",
			"/url/redirect",
			strings.NewReader(`{"short_url": "http://short.url/someslug"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusInternalServerError {
			t.Errorf("Received %d, expected %d", status, http.StatusInternalServerError)
		}
		expected := "Internal server error: Could not forward short URL http://short.url/someslug."
		if res.Body.String() != expected {
			t.Errorf("Received %s, expected %s", res.Body.String(), expected)
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 302 Found when successful", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: "http://original.url"}
		req, err := http.NewRequest(
			"POST",
			"/url/redirect",
			strings.NewReader(`{"short_url": "http://short.url/someslug"}`),
		)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusFound {
			t.Errorf("Received %d, expected %d", status, http.StatusFound)
		}
		if res.Header().Get("Location") != "http://original.url" {
			t.Errorf("Received %s, expected %s", res.Header().Get("Location"), "http://original.url")
		}
		App.UsService = OriginalUsService
	})
}

func TestHandleInternalUrlRedirect(t *testing.T) {
	t.Run("returns 405 Method Not Allowed when disallowed method is detected", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("POST", "/some-method", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Received %d, expected %d", status, http.StatusMethodNotAllowed)
		}
		if res.Body.String() != ResMethodNotAllowed {
			t.Errorf("Received %s, expected %s", res.Body.String(), ResMethodNotAllowed)
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 500 Internal Server Error when original url cannot be retrieved for short url", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: errors.New("failed"), shortUrl: "", originalUrl: ""}
		req, err := http.NewRequest("GET", "/some-method", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusInternalServerError {
			t.Errorf("Received %d, expected %d", status, http.StatusInternalServerError)
		}
		expected := fmt.Sprintf("Internal server error: Could not forward short URL %s/some-method.", App.EnvVars.InternalShortHost)
		if res.Body.String() != expected {
			t.Errorf("Received %s, expected %s", res.Body.String(), expected)
		}
		App.UsService = OriginalUsService
	})
	t.Run("returns 302 Found when successful", func(t *testing.T) {
		App.UsService = MockUsService{esIsLive: true, error: nil, shortUrl: "", originalUrl: "http://original.url"}
		req, err := http.NewRequest("GET", "/some-method", nil)
		if err != nil {
			t.Fatal(err)
		}
		res := httptest.NewRecorder()
		App.Routes.ServeHTTP(res, req)
		if status := res.Code; status != http.StatusFound {
			t.Errorf("Received %d, expected %d", status, http.StatusFound)
		}
		if res.Header().Get("Location") != "http://original.url" {
			t.Errorf("Received %s, expected %s", res.Header().Get("Location"), "http://original.url")
		}
		App.UsService = OriginalUsService
	})
}
