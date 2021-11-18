package main

import (
	"errors"
	"testing"
)

var callCount int

type MockPostgresDb struct {
	errors    []error
	id		  int
}

func (_ MockPostgresDb) Refresh() {
	return
}

func (m MockPostgresDb) connect() error {
	err := m.errors[callCount]
	callCount++
	return err
}

func (m MockPostgresDb) queryInt(_ string, _ ...interface{}) (int, error) {
	err := m.errors[callCount]
	callCount++
	return m.id, err
}

func (_ MockPostgresDb) close() {
	return
}

func TestKeyGenService_GetGeneratedKey(t *testing.T) {
	t.Run("returns error if key length is 0 or negative", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		_, err := kgSvc.GetGeneratedKey("some-source", 0)
		if err != ErrKeyLengthMustBePositive {
			t.Errorf("Received %s, expected %s", err, ErrKeyLengthMustBePositive)
		}
	})
	t.Run("returns error if cannot connect to db", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{errors.New("failed"), nil, nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		_, err := kgSvc.GetGeneratedKey("some-source", 8)
		if err != ErrCouldNotVerifySourceForKey {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotVerifySourceForKey)
		}
	})
	t.Run("returns error if source id cannot be retrieved", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, errors.New("failed"), nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		_, err := kgSvc.GetGeneratedKey("some-source", 8)
		if err != ErrCouldNotVerifySourceForKey {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotVerifySourceForKey)
		}
	})
	t.Run("returns error if key cannot be saved for source", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, errors.New("failed")}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		_, err := kgSvc.GetGeneratedKey("some-source", 8)
		if err != ErrCouldNotSaveKeyForSource {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotSaveKeyForSource)
		}
	})
	t.Run("returns key if successful", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, nil}, id: 123}
		kgSvc := NewKeyGenService(mockDb)
		key, err := kgSvc.GetGeneratedKey("some-source", 8)
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
		if len(key) < 1 {
			t.Errorf("Received nothing, expected string of non-zero length")
		}
	})
}

func TestKeyGenService_StoreCustomKey(t *testing.T) {
	t.Run("returns error if custom key is empty", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		err := kgSvc.StoreCustomKey("some-source", "")
		if err != ErrCustomKeyCannotBeEmpty {
			t.Errorf("Received %s, expected %s", err, ErrCustomKeyCannotBeEmpty)
		}
	})
	t.Run("returns error if cannot connect to db", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{errors.New("failed"), nil, nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		err := kgSvc.StoreCustomKey("some-source", "some-key")
		if err != ErrCouldNotVerifySourceForKey {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotVerifySourceForKey)
		}
	})
	t.Run("returns error if source id cannot be retrieved", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, errors.New("failed"), nil, nil}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		err := kgSvc.StoreCustomKey("some-source", "some-key")
		if err != ErrCouldNotVerifySourceForKey {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotVerifySourceForKey)
		}
	})
	t.Run("returns error if key cannot be saved for source", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, errors.New("failed")}, id: 0}
		kgSvc := NewKeyGenService(mockDb)
		err := kgSvc.StoreCustomKey("some-source", "some-key")
		if err != ErrCouldNotSaveKeyForSource {
			t.Errorf("Received %s, expected %s", err, ErrCouldNotSaveKeyForSource)
		}
	})
	t.Run("returns nil if successful", func(t *testing.T) {
		callCount = 0
		mockDb := MockPostgresDb{errors: []error{nil, nil, nil, nil}, id: 123}
		kgSvc := NewKeyGenService(mockDb)
		err := kgSvc.StoreCustomKey("some-source", "some-key")
		if err != nil {
			t.Errorf("Received %s, expected nil", err)
		}
	})
}