package main

import (
	"encoding/json"
	"errors"
	"testing"
)

type MockEsService struct {
	id string
	document Document
	error error
}

func (m MockEsService) PrintInfo() error {
	return m.error
}

func (m MockEsService) RefreshIndices(_ []string) error {
	return m.error
}

func (m MockEsService) IndexDocument(_ string, _ Document) (string, error) {
	return m.id, m.error
}

func (m MockEsService) GetDocumentById(_ string, _ string) (Document, error) {
	return m.document, m.error
}

type MockKgsService struct {
	key string
	error error
}

func (m MockKgsService) GenerateKey(_ string, _ int) (string, error) {
	return m.key, m.error
}

func (m MockKgsService) CreateNewKey(_ string, _ string) (string, error) {
	return m.key, m.error
}

func TestUrlShortenService_TestElasticsearchConnection(t *testing.T) {
	t.Run("returns false when connection test fails", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, errors.New("failed")}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		res := urlSvc.TestElasticsearchConnection()
		if res != false {
			t.Errorf("Received %t, expected %t", res, false)
		}
	})
	t.Run("returns true when successful", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		res := urlSvc.TestElasticsearchConnection()
		if res != true {
			t.Errorf("Received %t, expected %t", res, true)
		}
	})
}

func TestUrlShortenService_RefreshElasticsearchIndex(t *testing.T) {
	t.Run("returns error when indices refresh fails", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, errors.New("failed")}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		err := urlSvc.RefreshElasticsearchIndex()
		if err != ErrCouldNotRefreshElasticsearchIndex {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotRefreshElasticsearchIndex)
		}
	})
	t.Run("returns nil when successful", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		err := urlSvc.RefreshElasticsearchIndex()
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
	})
}

func TestUrlShortenService_ConstructShortUrlAndAssignToOriginalUrl(t *testing.T) {
	t.Run("returns error when construction fails", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", errors.New("failed")}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.ConstructShortUrlAndAssignToOriginalUrl(
			"http://some-url","http://shortho.st", "custom-slug", 0,
		)
		if err != ErrCouldNotConstructShortUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotConstructShortUrl)
		}
	})
	t.Run("returns error when assignment fails", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, errors.New("failed")}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.ConstructShortUrlAndAssignToOriginalUrl(
			"http://some-url","http://shortho.st", "custom-slug", 0,
		)
		if err != ErrCouldNotAssignShortUrlToOriginalUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotAssignShortUrlToOriginalUrl)
		}
	})
	t.Run("returns short url when successful", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"custom-slug", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		shortUrl, err := urlSvc.ConstructShortUrlAndAssignToOriginalUrl(
			"http://some-url","http://shortho.st", "custom-slug", 0,
		)
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
		if shortUrl != "http://shortho.st/custom-slug" {
			t.Errorf("Received %s, expected %s", shortUrl, "http://shortho.st/custom-slug")
		}
	})
}

func TestUrlShortenService_constructShortUrl(t *testing.T) {
	t.Run("returns error when new key cannot be created", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", errors.New("failed")}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.constructShortUrl("http://shortho.st", "custom-slug", 0)
		if err != ErrCouldNotCreateNewSlugForShortUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotCreateNewSlugForShortUrl)
		}
	})
	t.Run("returns error when new key cannot be generated", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", errors.New("failed")}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.constructShortUrl("http://shortho.st", "", 8)
		if err != ErrCouldNotGenerateNewSlugForShortUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotCreateNewSlugForShortUrl)
		}
	})
	t.Run("returns short url when successfully constructing with custom slug", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"custom-slug", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		shortUrl, err := urlSvc.constructShortUrl("http://shortho.st", "custom-slug", 0)
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
		if shortUrl != "http://shortho.st/custom-slug" {
			t.Errorf("Received %s, expected %s", shortUrl, "http://shortho.st/custom-slug")
		}
	})
	t.Run("returns short url when successfully constructing with generated slug", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"gen-slug", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		shortUrl, err := urlSvc.constructShortUrl("http://shortho.st", "", 8)
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
		if shortUrl != "http://shortho.st/gen-slug" {
			t.Errorf("Received %s, expected %s", shortUrl, "http://shortho.st/gen-slug")
		}
	})
}

func TestUrlShortenService_assignShortUrlToOriginalUrl(t *testing.T) {
	t.Run("returns error when document cannot be indexed", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, errors.New("failed")}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		err := urlSvc.assignShortUrlToOriginalUrl("http://some-url", "http://shrt-url")
		if err != ErrCouldNotStoreDocumentForShortUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotStoreDocumentForShortUrl)
		}
	})
	t.Run("returns nil when successful", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, nil}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		err := urlSvc.assignShortUrlToOriginalUrl("http://some-url", "http://shrt-url")
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
	})
}

func TestUrlShortenService_GetOriginalUrlForShortUrl(t *testing.T) {
	t.Run("returns error when document cannot be found", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{}, errors.New("not found")}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.GetOriginalUrlForShortUrl("http://shrt-url")
		if err != ErrCouldNotFindDocumentForShortUrl {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotFindDocumentForShortUrl)
		}
	})
	t.Run("returns error when document content JSON cannot be parsed", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{Id: "123", Content: json.RawMessage("{]")}, nil}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		_, err := urlSvc.GetOriginalUrlForShortUrl("http://shrt-url")
		if err != ErrCouldNotParseDocumentJson {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotParseDocumentJson)
		}
	})
	t.Run("returns nil when successful", func(t *testing.T) {
		mockEsService := MockEsService{"", Document{Id: "123", Content: json.RawMessage(`{"original_url": "http://some-url"}`)}, nil}
		mockKgsService := MockKgsService{"", nil}
		urlSvc := NewUrlShortenService("some-index", mockEsService, mockKgsService)
		url, err := urlSvc.GetOriginalUrlForShortUrl("http://shrt-url")
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
		if url != "http://some-url" {
			t.Errorf("Received %s, expected %s", url, "http://some-url")
		}
	})
}
