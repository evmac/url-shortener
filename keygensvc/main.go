package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// App

type KeyGenSvc struct {
	Flags struct {
		RefreshDb *bool
	}
	EnvVars struct {
		DbConnStr string
		MaxKeyLength int
		MinKeyLength int
		MinSourceNameLength int
	}
	Db PostgresDb
	Kg KeyGenService
}

var App KeyGenSvc

// Env Handlers

func HandleGetenvString(key string, required bool) string {
	envVar := os.Getenv(key)
	if required && envVar == "" {
		panic(fmt.Sprintf("Environment variable %s is not set", key))
	}
	return envVar
}

func HandleGetenvInt(key string, required bool) int {
	envVar := HandleGetenvString(key, required)
	intVar, err := strconv.Atoi(envVar)
	if err != nil {
		panic(fmt.Sprintf("Enviroment variable %s is not an integer", key))
	}
	return intVar
}

// Init

func init() {
	App.Flags.RefreshDb = flag.Bool(
		"refresh-database",
		false,
		"Runs database migration down and migration up.",
	)
	log.Print("Flags established")

	App.EnvVars.MaxKeyLength = HandleGetenvInt("MAXIMUM_KEY_LENGTH", true)
	App.EnvVars.MinKeyLength = HandleGetenvInt("MINIMUM_KEY_LENGTH", true)
	App.EnvVars.MinSourceNameLength = HandleGetenvInt("MINIMUM_SOURCE_NAME_LENGTH", true)
	App.EnvVars.DbConnStr    = HandleGetenvString("POSTGRES_CONNECTION_STRING", true)
	log.Print("Environment established")

	App.Db = NewPostgresDb(App.EnvVars.DbConnStr)
	App.Kg = NewKeyGenService(App.Db)
	log.Print("Service layer established")
}

// Main

func main() {
	// Parse and handle command-line flags
	flag.Parse()
	log.Print("Flags parsed, handling...")
	if App.Flags.RefreshDb != nil && *App.Flags.RefreshDb {
		App.Db.Refresh()
		log.Printf("Successfully refreshed database")
		return
	}

	// Instantiate routes and HTTP server
	http.HandleFunc("/key/generate", HandleGenerateKeyRequest)
	http.HandleFunc("/key/new", HandleNewKeyRequest)
	log.Print("Routes established, listening...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
