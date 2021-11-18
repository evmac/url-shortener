package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"math"
)

type KeyGenService interface {
	GetGeneratedKey(sourceName string, keyLength int) (string, error)
	StoreCustomKey(sourceName string, customKey string) error
}

type keyGenService struct {
	Db PostgresDb
}

func NewKeyGenService(db PostgresDb, ) KeyGenService {
	return &keyGenService{Db: db}
}

var (
	ErrSourceNameCannotBeEmpty    = errors.New("source name cannot be empty")
	ErrKeyLengthMustBePositive	  = errors.New("key length must be positive")
	ErrCustomKeyCannotBeEmpty     = errors.New("custom key cannot be empty")
	ErrCouldNotVerifySourceForKey = errors.New("could not verify source for key")
	ErrCouldNotSaveKeyForSource   = errors.New("could not save key for source")
	ErrCouldNotConnectToPostgres  = errors.New("could not connect to postgres db")
	ErrCouldNotRetrieveSourceId   = errors.New("could not retrieve source id")
	ErrCouldNotAddNewSource       = errors.New("could not add new source")
	ErrKeyAlreadyExists			  = errors.New("key already exists")
	ErrCouldNotSaveNewKey		  = errors.New("could not save new key")
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go

func (_ keyGenService) generateUniqueKey(keyLength int) string {
	buff := make([]byte, int(math.Ceil(float64(keyLength) / 1.33333333333)))
	_, _ = rand.Read(buff)
	return base64.RawURLEncoding.EncodeToString(buff)
}

func (kg keyGenService) GetGeneratedKey(sourceName string, keyLength int) (string, error) {
	if keyLength < 1 {
		return "", ErrKeyLengthMustBePositive
	}

	// Create source in DB if it does not exist
	sourceId, getErr := kg.getSourceId(sourceName)
	if getErr != nil {
		log.Printf(
			"Error getting source id for %s: %s", sourceName, getErr,
		)
		return "", ErrCouldNotVerifySourceForKey
	}

	// Generate new key
	key := kg.generateUniqueKey(keyLength)

	// Store key
	createErr := kg.createKey(sourceId, key)
	if createErr != nil {
		log.Printf(
			"Error saving key %s for %s: %s", key, sourceName, createErr,
		)
		return "", ErrCouldNotSaveKeyForSource
	}

	return key, nil
}

func (kg keyGenService) StoreCustomKey(sourceName string, customKey string) error {
	if customKey == "" {
		return ErrCustomKeyCannotBeEmpty
	}

	// Create source in DB if it does not exist
	sourceId, getErr := kg.getSourceId(sourceName)
	if getErr != nil {
		log.Printf(
			"Error getting source id for %s: %s", sourceName, getErr,
		)
		return ErrCouldNotVerifySourceForKey
	}

	// Store key
	createErr := kg.createKey(sourceId, customKey)
	if createErr != nil {
		log.Printf(
			"Error saving key %s for %s: %s", customKey, sourceName, createErr,
		)
		return ErrCouldNotSaveKeyForSource
	}

	return nil
}

func (kg keyGenService) getSourceId(sourceName string) (int, error) {
	var err error
	err = kg.Db.connect()
	if err != nil {
		log.Printf("Failed to connect to Postgres DB: %s", err)
		return -1, ErrCouldNotConnectToPostgres
	}
	defer kg.Db.close()

	var sourceId int
	sourceId, err = kg.Db.queryInt(
		"INSERT INTO sources (name) VALUES ($1) RETURNING id", sourceName,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			log.Printf("Source already exists for %s, retrieving id...", sourceName)
			sourceId, err = kg.Db.queryInt(
				"SELECT id FROM sources WHERE name = $1 AND is_active IS TRUE",
				sourceName,
			)
			if err != nil {
				log.Printf("Error retrieving source id: %s", err)
				return -1, ErrCouldNotRetrieveSourceId
			}
		} else {
			log.Printf("Error adding new source: %s", err)
			return -1, ErrCouldNotAddNewSource
		}
	} else {
		log.Printf("Inserted new source: id %d for %s", sourceId, sourceName)
	}

	log.Printf("Returning id %d for source %s", sourceId, sourceName)
	return sourceId, nil
}

func (kg keyGenService) createKey(sourceId int, key string) error {
	var err error
	err = kg.Db.connect()
	if err != nil {
		log.Printf("Error establishing connection to DB: %s", err)
		return ErrCouldNotConnectToPostgres
	}
	defer kg.Db.close()

	var keyId int
	keyId, err = kg.Db.queryInt(
		"INSERT INTO keys (raw_key, source_id) VALUES ($1, $2) RETURNING id",
		key,
		sourceId,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			log.Printf("Key %s already exists for source %d", key, sourceId)
			return ErrKeyAlreadyExists
		} else {
			log.Printf("Error inserting new key: %s", err)
			return ErrCouldNotSaveNewKey
		}
	}

	log.Printf(
		"Inserted new source: id %d for source %d and key %s", keyId, sourceId, key,
	)
	return nil
}
