package main

import (
  "log"
  "context"
  "os"
  "fmt"
  "github.com/gin-gonic/gin"
  "google.golang.org/api/gmail/v1"
  "github.com/joho/godotenv"
  "golang.org/x/oauth2"
  "google.golang.org/api/option"
)

// Add a new historyId to the buffer
func addToBuffer(historyBuffer *[]uint64, historyID uint64) {
  //NOTE: we use pointer types here because operating on a slice returns a new slice leaving the original unchanged.
  *historyBuffer = append(*historyBuffer, historyID) 
  if len(*historyBuffer) > 1 {
    *historyBuffer = (*historyBuffer)[1:] // Remove the first element
  }
}

// Get the oldest historyId from the buffer
func getOldestHistoryID(historyBuffer []uint64) uint64 {
  if len(historyBuffer) == 0 {
    return 0
  }
  return historyBuffer[0]
}

func main() {
  var historyBuffer []uint64 // Maintains older history IDs

  messageSet := make(map[string]bool) // To deduplicate messages


  // SetupGmailWatch()
  err := godotenv.Load(".env")
  if err != nil {
    fmt.Errorf("Error loading .env file: %v", err)
  }
  accessToken := os.Getenv("ACCESS_TOKEN")
  refreshToken := os.Getenv("REFRESH_TOKEN")
  clientID := os.Getenv("CLIENT_ID")
  clientSecret := os.Getenv("CLIENT_SECRET")

  if accessToken == "" || refreshToken == "" || clientID == "" || clientSecret == "" {
    fmt.Errorf("Missing required environment variables")
  }

  // OAuth2 Config
  config := &oauth2.Config{
    ClientID:     clientID,
    ClientSecret: clientSecret,
    Endpoint: oauth2.Endpoint{
      TokenURL: "https://oauth2.googleapis.com/token",
    },
  }

  token := &oauth2.Token{
    AccessToken:  accessToken,
    RefreshToken: refreshToken,
  }

  ctx := context.Background()
  client := config.Client(ctx, token)
  srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
  if err != nil {
    fmt.Errorf("Unable to create Gmail service: %v, %v", err, srv)
  }

  r := gin.Default()
  setupRoutes(r, srv, &historyBuffer, messageSet)
  if err := r.Run("0.0.0.0:3000"); err != nil {
    log.Fatalf("Error starting server: %v\n", err)
  }
}
