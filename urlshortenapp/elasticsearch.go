package main

// Generic module for REST interaction with Elasticsearch

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/esapi"
	es "github.com/elastic/go-elasticsearch/v7"
	"io"
	"log"
	"strings"
)

type EsService interface {
	PrintInfo() error
	RefreshIndices(indices []string) error
	IndexDocument(index string, document Document) (string, error)
	GetDocumentById(index string, id string) (Document, error)
}

type esService struct {
	EsClient *es.Client
	EsApi EsApi
}

type EsApi interface {
	Info(s *esService) (*esapi.Response, error)
	IndicesDelete(s *esService, indices []string) (*esapi.Response, error)
	IndicesCreate(s *esService, index string) (*esapi.Response, error)
	Index(s *esService, index string, json io.Reader, id string) (*esapi.Response, error)
	Get(s *esService, index string, id string) (*esapi.Response, error)
}

type esApi struct {}

func (_ *esApi) Info(s *esService) (*esapi.Response, error) {
	res, err := esapi.InfoRequest{}.Do(context.Background(), s.EsClient)
	return res, err
}

func (_ *esApi) IndicesDelete(s *esService, indices []string) (*esapi.Response, error) {
	res, err := esapi.IndicesDeleteRequest{Index: indices}.Do(context.Background(), s.EsClient)
	return res, err
}

func (_ *esApi) IndicesCreate(s *esService, index string) (*esapi.Response, error) {
	res, err := esapi.IndicesCreateRequest{Index: index}.Do(context.Background(), s.EsClient)
	return res, err
}

func (_ *esApi) Index(s *esService, index string, json io.Reader, id string) (*esapi.Response, error) {
	res, err := esapi.IndexRequest{
		Index: index,
		Body: json,
		DocumentID: id,
		Refresh: "true",
	}.Do(context.Background(), s.EsClient)
	return res, err
}

func (_ *esApi) Get(s *esService, index string, id string) (*esapi.Response, error) {
	res, err := esapi.GetRequest{
		Index: index,
		DocumentID: id,
	}.Do(context.Background(), s.EsClient)
	return res, err
}

func NewEsApi() EsApi {
	return &esApi{}
}

var (
	ErrEsClientNotInstantiated    = errors.New("elasticsearch Client not instantiated")
	ErrEsCouldNotFulfillRequest   = errors.New("elasticsearch could not fulfill request")
	ErrCouldNotParseResponseJson_ = errors.New("could not parse response json")
	ErrEsCouldNotDeleteIndices    = errors.New("elasticsearch could not delete indices")
	ErrEsCouldNotCreateIndex      = errors.New("elasticsearch could not create Index")
	ErrEsDoesNotContainDocument   = errors.New("elasticsearch does not contain document")
)

type Document struct {
	Id      string
	Content json.RawMessage
}

func NewEsService(esAddresses []string, esApi EsApi) (EsService, error) {
	esClient, clientErr := es.NewClient(es.Config{Addresses: esAddresses})
	if clientErr != nil {
		return nil, ErrEsClientNotInstantiated
	}
	return &esService{EsClient: esClient, EsApi: esApi}, nil
}

type infoResponseJson struct {
	Version struct {
		Number string `json:"number"`
	} `json:"version"`
}

func (s *esService) PrintInfo() error {
	// Make Info request
	httpResponse, err := s.EsApi.Info(s)
	if err != nil {
		log.Printf("Error printing info: %s", err)
		return ErrEsCouldNotFulfillRequest
	}
	defer httpResponse.Body.Close()

	// Parse response
	var responseJson infoResponseJson
	jsonErr := json.Unmarshal(
		parseRawJsonFromHttpBody(httpResponse.Body),
		&responseJson,
	)
	if jsonErr != nil {
		log.Printf("Error parsing the info response body: %s", jsonErr)
		return ErrCouldNotParseResponseJson_
	}

	// Print version info
	log.Print("Successfully connected to Elasticsearch cluster")
	log.Printf("Client: %s", es.Version)
	log.Printf("Server: %s", responseJson.Version.Number)

	return nil
}

func (s *esService) RefreshIndices(indices []string) error {
	// Make IndicesDelete request
	_, indicesDeleteErr := s.EsApi.IndicesDelete(s, indices)
	if indicesDeleteErr != nil {
		log.Printf("Error deleting indices: %s", indicesDeleteErr)
		return ErrEsCouldNotDeleteIndices
	}
	log.Printf("Deleted indices: [%s]", strings.Join(indices, ", "))

	// Make IndicesCreate requests
	for _, index := range indices {
		_, indicesCreateError := s.EsApi.IndicesCreate(s, index)
		if indicesCreateError != nil {
			log.Printf("Error creating index %s: %s", index, indicesCreateError)
			return ErrEsCouldNotCreateIndex
		}
		log.Printf("Created index: %s", index)
	}

	return nil
}

type indexResponseJson struct {
	Result  string `json:"result"`
	Id      string `json:"_id"`
	Version int    `json:"_version"`
}

func (s *esService) IndexDocument(index string, document Document) (string, error) {
	// Encode document content and construct Index request
	encodedContent, _ := document.Content.MarshalJSON()

	// Make Index request
	httpResponse, err := s.EsApi.Index(
		s, index, strings.NewReader(string(encodedContent)), document.Id,
	)
	if err != nil {
		log.Printf(
			"Error indexing document for id %s: %s", document.Id, err,
		)
		return "", ErrEsCouldNotFulfillRequest
	}
	defer httpResponse.Body.Close()
	log.Print("Received response from Elasticsearch for index request")

	// Parse response
	var responseJson indexResponseJson
	jsonErr := json.Unmarshal(
		parseRawJsonFromHttpBody(httpResponse.Body),
		&responseJson,
	)
	if jsonErr != nil {
		log.Printf("Error parsing the index response body: %s", jsonErr)
		return "", ErrCouldNotParseResponseJson_
	}
	log.Print("No errors in response to index request")

	// Return document Id
	log.Printf(
		"[%d] Response for index request parsed: %s; id=%s version=%d",
		httpResponse.StatusCode,
		responseJson.Result,
		responseJson.Id,
		responseJson.Version,
	)
	return responseJson.Id, nil
}

type getResponseJson struct {
	Found   bool   			`json:"found"`
	Id      string 			`json:"_id"`
	Version int   			`json:"_version"`
	Source  json.RawMessage `json:"_source"`
}

func (s *esService) GetDocumentById(index string, id string) (Document, error) {
	// Make Get request
	httpResponse, err := s.EsApi.Get(s, index, id)
	if err != nil {
		log.Printf("Error in get response: %s", err)
		return Document{}, ErrEsCouldNotFulfillRequest
	}
	defer httpResponse.Body.Close()
	log.Print("No errors in response to get request")

	// Parse response
	var responseJson getResponseJson
	jsonErr := json.Unmarshal(
		parseRawJsonFromHttpBody(httpResponse.Body),
		&responseJson,
	)
	if jsonErr != nil {
		log.Printf("Error parsing the get response body: %s", jsonErr)
		return Document{}, ErrCouldNotParseResponseJson_
	}
	log.Printf("[%d] Response to get request parsed", httpResponse.StatusCode)

	// Handle document not found
	if !responseJson.Found {
		log.Printf(
			"[%d] Document not found for id %s", httpResponse.StatusCode, id,
		)
		return Document{}, ErrEsDoesNotContainDocument
	}

	// Return document
	log.Printf(
		"[%d] Retrieved document for id %s", httpResponse.StatusCode, id,
	)
	return Document{
		Id:      responseJson.Id,
		Content: responseJson.Source,
	}, nil
}
