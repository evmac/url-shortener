package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// Utils

func parseRawJsonFromHttpBody(body io.ReadCloser) json.RawMessage {
	var buff bytes.Buffer
	_, _ = buff.ReadFrom(body)
	return json.RawMessage(buff.String())
}

type ValidationError string

type Validation struct {
	Errors []ValidationError
}

func (validation *Validation) Append(errorMessage string) {
	validation.Errors = append(validation.Errors, ValidationError(errorMessage))
}

func (validation *Validation) Fails() bool {
	return len(validation.Errors) > 0
}

// Response handlers

var (
	ResDefaultMessage 			= "Use a shortened link or use /url/shorten to shorten URLs."
	ResApplicationUnhealthy		= "Application not healthy"
	ResApplicationHealthy		= "Application healthy"
	ResMethodNotAllowed 		= "Method not allowed: see Allow header for allowed methods."
	ResCouldNotParseRequestJson = "Could not parse request JSON"
)

func handleCreated(w http.ResponseWriter, responseJson json.RawMessage) {
	log.Print("Returning 'Created' to caller")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(responseJson)
}

func handleFound(w http.ResponseWriter, redirectUrl string) {
	log.Print("Returning 'Found' to caller")
	w.Header().Set("Content-Type", "")
	w.Header().Set("Location", redirectUrl)
	w.WriteHeader(http.StatusFound)
}

func handleBadRequest(w http.ResponseWriter, responseJson json.RawMessage) {
	log.Print("Returning 'Bad Request' to caller")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(responseJson)
}

func handleMethodNotAllowed(w http.ResponseWriter, allowedMethods []string) {
	log.Print("Returning 'Method Not Allowed' to caller")
	w.Header().Set("Allow", strings.Join(allowedMethods, ","))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = fmt.Fprint(w, ResMethodNotAllowed)
}

func handleUnprocessableEntity(w http.ResponseWriter, message string) {
	log.Print("Returning 'Unprocessable Entity' to caller")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, _ = fmt.Fprintf(w, "Unprocessable entity: %s.", message)
}

func handleInternalServerError(w http.ResponseWriter, message string) {
	log.Print("Returning 'Internal Server Error' to caller")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = fmt.Fprintf(w, "Internal server error: %s.", message)
}

func handleServiceUnavailable(w http.ResponseWriter, message string) {
	log.Print("Returning 'Service Unavailable' to caller")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = fmt.Fprintf(w, "Service unavailable: %s.", message)
}

// Request handlers

func isMethodAllowed(method string, allowedMethods []string) bool {
	for _, allowedMethod := range allowedMethods {
		if method == allowedMethod {
			return true
		}
	}
	return false
}

func HandleIndexRequest(w http.ResponseWriter, _ *http.Request) {
	log.Print("/ hit")
	fmt.Fprint(w, ResDefaultMessage)
}

func HandleHealthcheckRequest(w http.ResponseWriter, _ *http.Request) {
	log.Print("/healthcheck hit")
	if !App.VerifyHealth() {
		handleServiceUnavailable(w, ResApplicationUnhealthy)
		return
	}
	_, _ = w.Write([]byte(ResApplicationHealthy))
}

type urlShortenRequestJson struct {
	OriginalUrl  string `json:"original_url"`
	ShortUrlHost string `json:"short_url_host"`
	CustomSlug   string `json:"custom_slug"`
	SlugLength   int    `json:"slug_length"`
}

func (r urlShortenRequestJson) Validate() Validation {
	var validation Validation

	// Validate original URL
	originalUrlTemplate, _ := regexp.Compile("^(http|https)://[a-zA-Z0-9\\.\\/\\?\\=\\_\\-]+$")
	if !originalUrlTemplate.MatchString(r.OriginalUrl) {
		validation.Append(
			fmt.Sprintf("Provided original URL is invalid: %s", r.OriginalUrl),
		)
	}

	// Validate short URL host
	if r.ShortUrlHost != "" {
		shortHostTemplate, _ := regexp.Compile("^(http|https)://[a-zA-Z0-9\\.]+$")
		if !shortHostTemplate.MatchString(r.ShortUrlHost) {
			validation.Append(
				fmt.Sprintf("Provided short host is invalid: %s", r.ShortUrlHost),
			)
		}
	}

	// Validate custom slug
	if r.CustomSlug != "" {
		if len(r.CustomSlug) < App.EnvVars.MinShortUrlPathLength ||
					len(r.CustomSlug) > App.EnvVars.MaxShortUrlPathLength {
			validation.Append(
				fmt.Sprintf(
					"Provided slug has incorrect length, minimum is %d and maximum is %d",
					App.EnvVars.MinShortUrlPathLength,
					App.EnvVars.MaxShortUrlPathLength,
				),
			)
		}
	}

	// Validate slug length
	if r.SlugLength > 0 {
		if r.SlugLength < App.EnvVars.MinShortUrlPathLength ||
					r.SlugLength > App.EnvVars.MaxShortUrlPathLength {
			validation.Append(
				fmt.Sprintf(
					"Requested slug length is too short, minimum is %d",
					App.EnvVars.MinShortUrlPathLength,
				),
			)
		}
	}

	return validation
}

type urlShortenResponseJson struct {
	OriginalUrl 	 string   		   `json:"original_url"`
	ShortUrl         string      	   `json:"short_url"`
	ValidationErrors []ValidationError `json:"validation_errors"`
}

func HandleUrlShortenRequest(w http.ResponseWriter, r *http.Request) {
	log.Print("/url/shorten hit")

	// Check method for validity
	allowedMethods := []string{http.MethodPost}
	if !isMethodAllowed(r.Method, allowedMethods) {
		handleMethodNotAllowed(w, allowedMethods)
		return
	}
	log.Printf("%s /url/shorten allowed", r.Method)

	// Parse request
	rawJson := parseRawJsonFromHttpBody(r.Body)
	var requestJson urlShortenRequestJson
	jsonUnmarshalErr := json.Unmarshal(rawJson, &requestJson)
	if jsonUnmarshalErr != nil {
		log.Printf(
			"Error parsing the Index document response body: %s", jsonUnmarshalErr,
		)
		handleUnprocessableEntity(w, ResCouldNotParseRequestJson)
		return
	}
	log.Print("Request JSON parsed")

	// Validate request
	validation := requestJson.Validate()
	log.Print("Request JSON validated")

	// Construct response JSON
	responseJson := urlShortenResponseJson{
		OriginalUrl: requestJson.OriginalUrl,
		ShortUrl: "",  // Will update later if successful
		ValidationErrors: validation.Errors,
	}

	// Short-circuit if we have validation errors
	if validation.Fails() {
		log.Print("Validation failed...")

		// Encode response JSON
		encodedJson, _ := json.Marshal(responseJson)

		// Send response
		handleBadRequest(w, encodedJson)
		return
	}

	originalUrl  := requestJson.OriginalUrl
	shortUrlHost := requestJson.ShortUrlHost
	customSlug   := requestJson.CustomSlug
	slugLength   := requestJson.SlugLength

	// Provide defaults if we validate request
	if shortUrlHost == "" {
		shortUrlHost = App.EnvVars.InternalShortHost
	}
	if slugLength <= 0 {
		slugLength = App.EnvVars.MinShortUrlPathLength
	}

	// Construct and assign short URL
	log.Print("Constructing and assigning short URL...")
	shortUrl, shortenErr := App.UsService.ConstructShortUrlAndAssignToOriginalUrl(
		originalUrl, shortUrlHost, customSlug, slugLength,
	)
	if shortenErr != nil {
		log.Printf("Unable to construct short URL for %s: %s", originalUrl, shortenErr)
		handleInternalServerError(
			w,
			fmt.Sprintf("Could not shorten URL %s", originalUrl),
		)
		return
	}
	log.Print("Constructed and assigned short URL")

	// Encode response JSON
	responseJson.ShortUrl = shortUrl
	encodedJson, _ := json.Marshal(responseJson)
	log.Print("Response encoded")

	// Send response
	handleCreated(w, encodedJson)
}

type urlRedirectExternalRequestJson struct {
	ShortUrl string `json:"short_url"`
}

func (r urlRedirectExternalRequestJson) Validate() Validation {
	var validation Validation
	shortUrlTemplate, _ := regexp.Compile(
		"^(http|https)://[a-zA-Z0-9\\.]+/[a-zA-Z0-9\\-_]+$",
	)
	if !shortUrlTemplate.MatchString(r.ShortUrl) {
		validation.Append(
			fmt.Sprintf("Provided short URL is invalid: %s", r.ShortUrl),
		)
	}
	return validation
}

type urlRedirectExternalResponseJson struct {
	ValidationErrors []ValidationError `json:"validation_errors"`
}

func HandleExternalUrlRedirect(w http.ResponseWriter, r *http.Request) {
	log.Print("/url/redirect hit")

	// Check method for validity
	allowedMethods := []string{http.MethodPost}
	if !isMethodAllowed(r.Method, allowedMethods) {
		handleMethodNotAllowed(w, allowedMethods)
		return
	}

	// Parse request
	rawJson := parseRawJsonFromHttpBody(r.Body)
	var requestJson urlRedirectExternalRequestJson
	jsonErr := json.Unmarshal(rawJson, &requestJson)
	if jsonErr != nil {
		log.Printf(
			"Error parsing the URL redirect request JSON: %s", jsonErr,
		)
		handleUnprocessableEntity(w, ResCouldNotParseRequestJson)
		return
	}

	// Validate request
	validation := requestJson.Validate()

	// Construct response JSON
	responseJson := urlRedirectExternalResponseJson{
		ValidationErrors: validation.Errors,
	}

	// Short-circuit if we have validation errors
	if validation.Fails() {
		// Encode response JSON
		encodedJson, _ := json.Marshal(responseJson)

		// Send response
		handleBadRequest(w, encodedJson)
		return
	}

	// Get original URL
	originalUrl, getErr := App.UsService.GetOriginalUrlForShortUrl(
		requestJson.ShortUrl,
	)
	if getErr != nil {
		log.Printf("Error getting original URL for short URL %s", requestJson.ShortUrl)
		handleInternalServerError(
			w,
			fmt.Sprintf("Could not forward short URL %s", requestJson.ShortUrl),
		)
		return
	}

	// Redirect to original URL
	log.Printf("Forwarding %s to %s", requestJson.ShortUrl, originalUrl)
	handleFound(w, originalUrl)
}

func HandleInternalUrlRedirect(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s hit", r.URL.Path)

	// Check method for validity
	allowedMethods := []string{http.MethodGet}
	if !isMethodAllowed(r.Method, allowedMethods) {
		handleMethodNotAllowed(w, allowedMethods)
		return
	}

	shortUrl := fmt.Sprintf("%s%s", App.EnvVars.InternalShortHost, r.URL.Path)

	// Get original URL
	originalUrl, err := App.UsService.GetOriginalUrlForShortUrl(shortUrl)
	if err != nil {
		log.Printf("Error getting original URL for short URL %s", shortUrl)
		handleInternalServerError(
			w,
			fmt.Sprintf("Could not forward short URL %s", shortUrl),
		)
		return
	}

	// Redirect to original URL
	log.Printf("Forwarding %s to %s", shortUrl, originalUrl)
	handleFound(w, originalUrl)
}
