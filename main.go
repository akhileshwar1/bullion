package main

import (
  "log"
  "context"
  "os"
  "time"
  "github.com/gin-gonic/gin"
  "google.golang.org/api/gmail/v1"
  "github.com/joho/godotenv"
  "golang.org/x/oauth2"
  "google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// LoggingTokenSource is a custom TokenSource wrapper that logs refresh events.
type LoggingTokenSource struct {
  source oauth2.TokenSource
}

func (l *LoggingTokenSource) Token() (*oauth2.Token, error) {
  // Fetch the token (this triggers the refresh if the token is expired)
  newToken, err := l.source.Token()
  if err != nil {
    return nil, err
  }

  // Log when the token is refreshed
  if newToken.Valid() && newToken.AccessToken != "" {
    log.Println(newToken)
    log.Printf("Refreshed Access Token: %s\n", newToken.AccessToken)
  }
  return newToken, nil
}

func main() {
  err := godotenv.Load(".env")
  if err != nil {
    log.Printf("Error loading .env from default location: %v", err)
    // Try loading it from /root/.env inside the container
    err = godotenv.Load("/root/.env")
    if err != nil {
      log.Fatalf("Error loading .env from /root/.env: %v", err)
    }
  }

  clientID := os.Getenv("CLIENT_ID")
  clientSecret := os.Getenv("CLIENT_SECRET")
  refreshToken := os.Getenv("REFRESH_TOKEN")
  accessToken := os.Getenv("ACCESS_TOKEN")
  log.Println(accessToken)

  if clientID == "" || clientSecret == "" || refreshToken == "" {
    log.Fatalf("Missing required environment variables")
  }

  // OAuth2 Config
  config := &oauth2.Config{
    ClientID:     clientID,
    ClientSecret: clientSecret,
    Endpoint: oauth2.Endpoint{
      TokenURL: "https://oauth2.googleapis.com/token",
    },
    Scopes: []string{
      "https://www.googleapis.com/auth/gmail.readonly",
      "https://www.googleapis.com/auth/spreadsheets",
      "https://www.googleapis.com/auth/pubsub",
    },
  }

  // Create an initial token object
  token := &oauth2.Token{
    AccessToken:  accessToken,
    RefreshToken: refreshToken,
    Expiry:       time.Now().Add(-1 * time.Hour), //NOTE: this forces token expiry after an hour prompting the token source to refresh the access token.
  }

  // Wrap the token source with a LoggingTokenSource
  ctx := context.Background()
  baseTokenSource := config.TokenSource(ctx, token)
  loggingTokenSource := &LoggingTokenSource{source: baseTokenSource}

  // Use the custom LoggingTokenSource to create an HTTP client
  client := oauth2.NewClient(ctx, loggingTokenSource)

  // Initialize Gmail service
  gsrv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
  if err != nil {
    log.Fatalf("Unable to create Gmail service: %v", err)
  }

  // Initialize Sheets service
  ssrv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
  if err != nil {
    log.Fatalf("Unable to create Sheets service: %v", err)
  }

  ch := make(chan uint64, 100)
  go processNotifs(ch, gsrv, ssrv, "me")

  // Set up Gin routes and start the server
  r := gin.Default()
  setupRoutes(r, gsrv, ssrv, ch)
  if err := r.Run("0.0.0.0:3000"); err != nil {
    log.Fatalf("Error starting server: %v", err)
  }
}

func processNotifs(ch <-chan uint64, gsrv *gmail.Service, ssrv *sheets.Service, user string) {
  var prev uint64 = 0
  for data := range ch {
    if prev == 0 {
      prev = data
    } else {
      processHistory(gsrv, ssrv, user, prev)
      prev = data
    }
  }
}
