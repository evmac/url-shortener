package main

import (
    "errors"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "regexp"
    "strconv"
    "strings"
    "time"
)

// App

type UrlShortenApp struct {
    Flags struct {
        RefreshIndex *bool
    }
    EnvVars struct {
        EsAddresses           string
        EsIndex               string
        InitMaxAttempts       int
        InitWaitInSeconds     int
        KgsUrl                string
        InternalShortHost     string
        MinShortUrlPathLength int
        MaxShortUrlPathLength int
    }
    Routes    *Routes
    UsService UrlShortenService
}

var App UrlShortenApp

// Env Handlers

func HandleGetenvString(key string) string {
    envVar := os.Getenv(key)
    if envVar == "" {
        panic(fmt.Sprintf("Environment variable %s is not set", key))
    }
    return envVar
}

func HandleGetenvInt(key string) int {
    envVar := HandleGetenvString(key)
    intVar, err := strconv.Atoi(envVar)
    if err != nil {
        panic(fmt.Sprintf("Enviroment variable %s is not an integer", key))
    }
    return intVar
}


// Routes
// https://stackoverflow.com/questions/6564558/wildcards-in-the-pattern-for-http-handlefunc

type RouteHandler struct {
    Pattern *regexp.Regexp
    Func    http.Handler
}

type Routes struct {
    Handlers []*RouteHandler
}

func (routes *Routes) HandleFunc(
    pattern *regexp.Regexp, handler func(w http.ResponseWriter, r *http.Request),
) {
    routes.Handlers = append(
        routes.Handlers, &RouteHandler{Pattern: pattern, Func: http.HandlerFunc(handler)},
    )
}

func (routes *Routes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    for _, handler := range routes.Handlers {
        if handler.Pattern.MatchString(r.URL.Path) {
            log.Printf("Matched route %s", r.URL.Path)
            handler.Func.ServeHTTP(w, r)
            return
        }
    }

    log.Printf("Did not match any routes for %s", r.URL.Path)
    http.NotFound(w, r)
}

func (routes Routes) Define() *Routes {
    // Necessary as net/http cannot handle wildcard routes.
    // Use regex module to match routes, then handle internally.
    // Serve this definition on "/" - it matches all unknown routes.

    // Match index route
    indexRoute, _ := regexp.Compile("^/$")
    // Match healthcheck route
    healthcheckRoute, _ := regexp.Compile("^/healthcheck$")
    // Match URL-shorten route
    urlShortenRoute, _ := regexp.Compile("^/url/shorten$")
    // Match URL external redirect route
    urlRedirectExternalRoute, _ := regexp.Compile("^/url/redirect$")
    // Match everything else recognizable as an internal short URL
    urlRedirectInternalRoute, _ := regexp.Compile("^/[a-zA-Z0-9\\-_]+$")

    routes.HandleFunc(indexRoute, HandleIndexRequest)
    routes.HandleFunc(healthcheckRoute, HandleHealthcheckRequest)
    routes.HandleFunc(urlShortenRoute, HandleUrlShortenRequest)
    routes.HandleFunc(urlRedirectExternalRoute, HandleExternalUrlRedirect)
    routes.HandleFunc(urlRedirectInternalRoute, HandleInternalUrlRedirect)

    return &routes
}

func (a UrlShortenApp) VerifyHealth() bool {
    healthy := true
    log.Print("Running healthcheck...")
    healthy = healthy && a.UsService.TestElasticsearchConnection()
    return healthy
}

// Init

func init() {
    App.Flags.RefreshIndex = flag.Bool(
        "refresh-index",
        false,
        "Deletes and recreates Elasticsearch index.",
    )
    log.Printf("Flags established")

    App.EnvVars.EsAddresses           = HandleGetenvString("ELASTICSEARCH_ADDRESSES")
    App.EnvVars.EsIndex               = HandleGetenvString("ELASTICSEARCH_INDEX")
    App.EnvVars.InitMaxAttempts       = HandleGetenvInt("INIT_MAXIMUM_ATTEMPTS")
    App.EnvVars.InitWaitInSeconds     = HandleGetenvInt("INIT_WAIT_IN_SECONDS")
    App.EnvVars.KgsUrl                = HandleGetenvString("KEYGENSVC_URL")
    App.EnvVars.InternalShortHost     = HandleGetenvString("INTERNAL_SHORT_HOST")
    App.EnvVars.MinShortUrlPathLength = HandleGetenvInt("MINIMUM_SHORT_URL_PATH_LENGTH")
    App.EnvVars.MaxShortUrlPathLength = HandleGetenvInt("MAXIMUM_SHORT_URL_PATH_LENGTH")
    log.Print("Environment variables established")

    App.Routes = Routes{}.Define()
    log.Print("Routes defined")

    // Instantiate Elasticsearch service
    esSvc, esErr := NewEsService(
        strings.Split(App.EnvVars.EsAddresses, ","), NewEsApi(),
    )
    if esErr != nil {
        log.Printf("Error instantiating Elasticsearch service: %s", esErr)
        log.Fatal(errors.New("could not instantiate elasticsearch service"))
    }

    // Instantiate keygensvc service
    kgsSvc, kgsErr := NewKgsService(NewKgsClient(App.EnvVars.KgsUrl))
    if kgsErr != nil {
        log.Printf("Error instantiating keygensvc service: %s", kgsErr)
        log.Fatal(errors.New("could not instantiate keygensvc service"))
    }

    // Attach UrlShortenService to app
    App.UsService = NewUrlShortenService(App.EnvVars.EsIndex, esSvc, kgsSvc)
    log.Print("Service layer established")
}

// Main

func main() {
    // Run healthcheck on startup.
    // Necessary as Elasticsearch can take half a minute or more to start up,
    // and we want to wait until it's live before we begin serving routes.
    // It's also required to handle the --refresh-index flag.
    attempts := 0
    startTime := time.Now()
    for {
        if attempts == App.EnvVars.InitMaxAttempts {
            // Hard fail when we can't verify in a reasonable amount of time
            log.Fatal("Could not verify app health")
        }

        log.Printf("Verifying app health: attempt %d...", attempts + 1)

        // Check app health
        if !App.VerifyHealth() {
            // On failure, wait and try again
            waitInSeconds := time.Duration(App.EnvVars.InitWaitInSeconds) * time.Second
            log.Printf("Retrying in %s...", waitInSeconds.String())
            time.Sleep(waitInSeconds)
            attempts++
            continue
        }

        totalTime := time.Since(startTime)
        log.Printf("App health verified, took %s", totalTime.String())
        break
    }

    // Parse and handle command-line flags
    flag.Parse()
    log.Print("Flags parsed, handling...")
    if App.Flags.RefreshIndex != nil && *App.Flags.RefreshIndex {
        if err := App.UsService.RefreshElasticsearchIndex(); err != nil {
            log.Printf("Error while refreshing Elasticsearch index: %s", err)
        }
        return
    }

    // Instantiate HTTP server
    http.HandleFunc("/", App.Routes.ServeHTTP)
    log.Print("Routes established, listening...")
    log.Fatal(http.ListenAndServe(":80", nil))
}
