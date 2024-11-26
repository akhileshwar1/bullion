package main

import (
	"context"
	"log"
	"os"
  "fmt"
  "strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func SetupGmailWatch() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	accessToken := os.Getenv("ACCESS_TOKEN")
	refreshToken := os.Getenv("REFRESH_TOKEN")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	topicName := os.Getenv("TOPIC_NAME")

	if accessToken == "" || refreshToken == "" || clientID == "" || clientSecret == "" || topicName == "" {
		log.Fatalf("Missing required environment variables")
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

	// Check if the access token is valid
	client := config.Client(ctx, token)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Failed to create Gmail service: %v", err)
	}

	// Test the token by listing Gmail labels
	_, err = srv.Users.Labels.List("me").Do()
	if err != nil && strings.Contains(err.Error(), "401") { // Unauthorized error, token is likely invalid
		log.Println("Access token is invalid or expired. Refreshing token...")
		tokenSource := config.TokenSource(ctx, token)
		newToken, tokenErr := tokenSource.Token()
		if tokenErr != nil {
			log.Fatalf("Failed to refresh token: %v", tokenErr)
		}

		// Update the .env file with the new access token
		err := updateEnvFile("../.env", "ACCESS_TOKEN", newToken.AccessToken)
		if err != nil {
			log.Fatalf("Failed to update .env file: %v", err)
		}

		// Use the refreshed token
		client = config.Client(ctx, newToken)
		srv, err = gmail.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Failed to create Gmail service with refreshed token: %v", err)
		}
	} else if err != nil {
		log.Fatalf("Failed to validate access token: %v", err)
	}

	// Proceed with the watch request
	watchReq := &gmail.WatchRequest{
		LabelIds:  []string{"INBOX"},
		TopicName: topicName,
	}

	resp, err := srv.Users.Watch("me", watchReq).Do()
	if err != nil {
		log.Printf("Watch request failed: %v", err)
	} else {
		log.Printf("Watch request succeeded: %+v", resp)
	}
}

// Function to update the .env file
func updateEnvFile(filePath, key, value string) error {
	envMap, err := godotenv.Read(filePath)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %v", err)
	}

	envMap[key] = value

	err = godotenv.Write(envMap, filePath)
	if err != nil {
		return fmt.Errorf("failed to write to .env file: %v", err)
	}

	log.Printf("Updated .env file: %s = %s", key, value)
	return nil
}
