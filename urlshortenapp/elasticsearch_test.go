package main

import (
	"encoding/json"
	"errors"
	"github.com/elastic/go-elasticsearch/esapi"
	"io"
	"net/http"
	"strings"
	"testing"
)

type MockEsApi struct {
	statusCodes []int
	errors      []error
	bodies      []io.ReadCloser
}

func (m MockEsApi) Info(_ *esService) (*esapi.Response, error) {
	return &esapi.Response{StatusCode: m.statusCodes[0], Body: m.bodies[0]}, m.errors[0]
}

func (m MockEsApi) IndicesDelete(_ *esService, _ []string) (*esapi.Response, error) {
	return &esapi.Response{StatusCode: m.statusCodes[0], Body: m.bodies[0]}, m.errors[0]
}

func (m MockEsApi) IndicesCreate(_ *esService, _ string) (*esapi.Response, error) {
	return &esapi.Response{StatusCode: m.statusCodes[1], Body: m.bodies[1]}, m.errors[1]
}

func (m MockEsApi) Index(_ *esService, _ string, _ io.Reader, _ string) (*esapi.Response, error) {
	return &esapi.Response{StatusCode: m.statusCodes[0], Body: m.bodies[0]}, m.errors[0]
}

func (m MockEsApi) Get(_ *esService, _ string, _ string) (*esapi.Response, error) {
	return &esapi.Response{StatusCode: m.statusCodes[0], Body: m.bodies[0]}, m.errors[0]
}

func TestEsService_PrintInfo(t *testing.T) {
	t.Run("returns error when ES API Info call fails", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(""))},
			errors:      []error{errors.New("failed")},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		infoErr := esSvc.PrintInfo()
		if infoErr != ErrEsCouldNotFulfillRequest {
			t.Errorf("Received %s, expected %s", infoErr, ErrEsCouldNotFulfillRequest)
		}
	})
	t.Run("returns error when response json cannot be parsed", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("{]"))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		infoErr := esSvc.PrintInfo()
		if infoErr != ErrCouldNotParseResponseJson_ {
			t.Errorf("Received %s, expected %s", infoErr, ErrCouldNotParseResponseJson_)
		}
	})
	t.Run("returns nil when successful", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(`{"version": {"number": "123"}}`))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		infoErr := esSvc.PrintInfo()
		if infoErr != nil {
			t.Errorf("Received %s, expected nil", infoErr)
		}
	})
}

func TestEsService_RefreshIndices(t *testing.T) {
	t.Run("returns error when ES API IndicesDelete call fails", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0, 0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("")), io.NopCloser(strings.NewReader(""))},
			errors:      []error{errors.New("failed"), nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		refreshErr := esSvc.RefreshIndices([]string{"some-index"})
		if refreshErr != ErrEsCouldNotDeleteIndices {
			t.Errorf("Received %s, expected %s", refreshErr, ErrEsCouldNotDeleteIndices)
		}
	})
	t.Run("returns error when ES API IndicesCreate call fails", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0, 0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("")), io.NopCloser(strings.NewReader(""))},
			errors:      []error{nil, errors.New("failed")},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		refreshErr := esSvc.RefreshIndices([]string{"some-index"})
		if refreshErr != ErrEsCouldNotCreateIndex {
			t.Errorf("Received %s, expected %s", refreshErr, ErrEsCouldNotCreateIndex)
		}
	})
	t.Run("returns nil when successful", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0, 0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("")), io.NopCloser(strings.NewReader(""))},
			errors:      []error{nil, nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		refreshErr := esSvc.RefreshIndices([]string{"some-index"})
		if refreshErr != nil {
			t.Errorf("Received %s, expected nil", refreshErr)
		}
	})
}

func TestEsService_IndexDocument(t *testing.T) {
	t.Run("returns error when ES API Index call fails", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(""))},
			errors:      []error{errors.New("failed")},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		_, indexErr := esSvc.IndexDocument("some-index", Document{Id: "123", Content: json.RawMessage("{}")})
		if indexErr != ErrEsCouldNotFulfillRequest {
			t.Errorf("Received %s, expected %s", indexErr, ErrEsCouldNotFulfillRequest)
		}
	})
	t.Run("returns error when response JSON cannot be parsed", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("{]"))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		_, indexErr := esSvc.IndexDocument("some-index", Document{Id: "123", Content: json.RawMessage("{}")})
		if indexErr != ErrCouldNotParseResponseJson_ {
			t.Errorf("Received %s, expected %s", indexErr, ErrCouldNotParseResponseJson_)
		}
	})
	t.Run("returns id when successful", func(t *testing.T) {
		resJson := `{"result": "created", "_id": "123", "_version": 1}`
		mockEsApi := MockEsApi{
			statusCodes: []int{http.StatusOK},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(resJson))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		id, indexErr := esSvc.IndexDocument("some-index", Document{Id: "123", Content: json.RawMessage("{}")})
		if indexErr != nil {
			t.Errorf("Received %s, expected %s", indexErr, ErrCouldNotParseResponseJson_)
		}
		if id != "123" {
			t.Errorf("Received %s, expected %s", id, "123")
		}
	})
}

func TestEsService_GetDocumentById(t *testing.T) {
	t.Run("returns error when ES API Get call fails", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(""))},
			errors:      []error{errors.New("failed")},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		_, getErr := esSvc.GetDocumentById("some-index", "123")
		if getErr != ErrEsCouldNotFulfillRequest {
			t.Errorf("Received %s, expected %s", getErr, ErrEsCouldNotFulfillRequest)
		}
	})
	t.Run("returns error when response JSON cannot be parsed", func(t *testing.T) {
		mockEsApi := MockEsApi{
			statusCodes: []int{0},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader("{]"))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		_, getErr := esSvc.GetDocumentById("some-index", "123")
		if getErr != ErrCouldNotParseResponseJson_ {
			t.Errorf("Received %s, expected %s", getErr, ErrCouldNotParseResponseJson_)
		}
	})
	t.Run("returns error when document is not found", func(t *testing.T) {
		resJson := `{"found": false}`
		mockEsApi := MockEsApi{
			statusCodes: []int{http.StatusOK},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(resJson))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		_, getErr := esSvc.GetDocumentById("some-index", "123")
		if getErr != ErrEsDoesNotContainDocument {
			t.Errorf("Received %s, expected %s", getErr, ErrEsDoesNotContainDocument)
		}
	})
	t.Run("returns document when document is found", func(t *testing.T) {
		resJson := `{"found": true, "_id": "123", "_version": 1, "_source": "{}"}`
		mockEsApi := MockEsApi{
			statusCodes: []int{http.StatusOK},
			bodies:      []io.ReadCloser{io.NopCloser(strings.NewReader(resJson))},
			errors:      []error{nil},
		}
		esSvc, _ := NewEsService([]string{}, mockEsApi)
		doc, getErr := esSvc.GetDocumentById("some-index", "123")
		if getErr != nil {
			t.Errorf("Received %s, expected nil", getErr)
		}
		if doc.Id != "123" {
			t.Errorf("Received %s, expected %s", doc.Id, "123")
		}
	})
}
