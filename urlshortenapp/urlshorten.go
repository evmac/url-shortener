package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

type UrlShortenService interface {
	TestElasticsearchConnection() bool
	RefreshElasticsearchIndex() error
	ConstructShortUrlAndAssignToOriginalUrl(originalUrl string, shortHost string, customSlug string, slugLength int) (string, error)
	constructShortUrl(shortHost string, customSlug string, slugLength int) (string, error)
	assignShortUrlToOriginalUrl(url string, shortUrl string) error
	GetOriginalUrlForShortUrl(shortUrl string) (string, error)
}

type urlShortenService struct {
	EsIndex    string
	EsService  EsService
	KgsService KgsService
}

func NewUrlShortenService(
	esIndex string, esService EsService, kgsService KgsService,
) UrlShortenService {
	return &urlShortenService{
		EsIndex: esIndex,
		EsService: esService,
		KgsService: kgsService,
	}
}

var (
	ErrCouldNotRefreshElasticsearchIndex   = errors.New("could not refresh elasticsearch Index")
	ErrCouldNotConstructShortUrl		   = errors.New("could not construct short url")
	ErrCouldNotAssignShortUrlToOriginalUrl = errors.New("could not assign short url to original url")
	ErrCouldNotCreateNewSlugForShortUrl    = errors.New("could not create new slug for short url")
	ErrCouldNotGenerateNewSlugForShortUrl  = errors.New("could not generate new slug for short url")
	ErrCouldNotConstructDocumentJson       = errors.New("could not construct document content json")
	ErrCouldNotStoreDocumentForShortUrl    = errors.New("could not store document for url")
	ErrCouldNotFindDocumentForShortUrl     = errors.New("could not find document for short url")
	ErrCouldNotParseDocumentJson		   = errors.New("could not parse document content json")
)

func (s urlShortenService) TestElasticsearchConnection() bool {
	if infoErr := s.EsService.PrintInfo(); infoErr != nil {
		log.Printf("Error testing Elasticsearch connection: %s", infoErr)
		return false
	}
	return true
}

func (s urlShortenService) RefreshElasticsearchIndex() error {
	// Refresh Elasticsearch Index
	if refreshErr := s.EsService.RefreshIndices([]string{s.EsIndex}); refreshErr != nil {
		log.Printf("Error refreshing Elasticsearch index: %s", refreshErr)
		return ErrCouldNotRefreshElasticsearchIndex
	}
	log.Printf("Successfully refreshed Elasticsearch index %s", s.EsIndex)
	return nil
}

func (s urlShortenService) ConstructShortUrlAndAssignToOriginalUrl(
	originalUrl string, shortHost string, customSlug string, slugLength int,
) (string, error) {
	// Construct short URL
	shortUrl, constructErr := s.constructShortUrl(
		shortHost, customSlug, slugLength,
	)
	if constructErr != nil {
		log.Printf("Unable to construct short URL for %s: %s", originalUrl, constructErr)
		return "", ErrCouldNotConstructShortUrl
	}

	// Assign short URL
	assignErr := s.assignShortUrlToOriginalUrl(originalUrl, shortUrl)
	if assignErr != nil {
		log.Printf(
			"Unable to assign short URL %s to %s: %s", shortUrl, originalUrl, assignErr,
		)
		return "", ErrCouldNotAssignShortUrlToOriginalUrl
	}

	return shortUrl, nil
}

func (s urlShortenService) constructShortUrl(shortHost string, customSlug string, slugLength int) (string, error) {
	var slug string
	var err error
	if customSlug != "" {
		// Create new slug for short URL
		log.Print("Creating new slug for short URL...")
		slug, err = s.KgsService.CreateNewKey(shortHost, customSlug)
		if err != nil {
			log.Printf(
				"Error creating new slug to construct short URL for host %s: %s", shortHost, err,
			)
			return "", ErrCouldNotCreateNewSlugForShortUrl
		}
		log.Printf("New slug created: %s", slug)
	} else {
		// Generate new slug for short URL
		log.Print("Retrieving new slug for short URL...")
		slug, err = s.KgsService.GenerateKey(shortHost, slugLength)
		if err != nil {
			log.Printf(
				"Error generating new slug to construct short URL for host %s: %s", shortHost, err,
			)
			return "", ErrCouldNotGenerateNewSlugForShortUrl
		}
		log.Printf("New slug retrieved: %s", slug)
	}

	// Return constructed short URL
	return fmt.Sprintf("%s/%s", shortHost, slug), nil
}

type urlDocumentContent struct {
	OriginalUrl string `json:"original_url"`
	ShortUrl    string `json:"short_url"`
}

func (s urlShortenService) assignShortUrlToOriginalUrl(url string, shortUrl string) error {
	// Construct new document
	shortUrlHash := md5.Sum([]byte(shortUrl))
	newId := hex.EncodeToString(shortUrlHash[:])

	content, _ := json.Marshal(
		urlDocumentContent{OriginalUrl: url, ShortUrl: shortUrl},
	)
	document := Document{Id: newId, Content: content}
	log.Printf("Constructed new document: hash %s", newId)

	// Store document in Elasticsearch
	id, indexErr := s.EsService.IndexDocument(s.EsIndex, document)
	if indexErr != nil {
		log.Printf(
			"Error indexing given URL %s: %s; hash=%s", url, indexErr, newId,
		)
		return ErrCouldNotStoreDocumentForShortUrl
	}
	log.Printf("Indexed document: %s", id)

	return nil
}

func (s urlShortenService) GetOriginalUrlForShortUrl(shortUrl string) (string, error) {
	// Get hash id for short URL
	shortUrlHash := md5.Sum([]byte(shortUrl))
	id := hex.EncodeToString(shortUrlHash[:])

	// Fetch document from Elasticsearch
	document, getErr := s.EsService.GetDocumentById(App.EnvVars.EsIndex, id)
	if getErr != nil {
		log.Printf("Error finding URL for given short URL %s: %s", shortUrl, getErr)
		return "", ErrCouldNotFindDocumentForShortUrl
	}

	// Parse document content for URL
	content := urlDocumentContent{}
	parseContentErr := json.Unmarshal(document.Content, &content)
	if parseContentErr != nil {
		log.Printf("Error parsing document content for short URL: %s", shortUrl)
		return "", ErrCouldNotParseDocumentJson
	}

	// Return document URL
	return content.OriginalUrl, nil
}
