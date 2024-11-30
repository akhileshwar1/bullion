package main

import (
	"fmt"
  "strings"
	"context"
  "os"
	"net/http"
  "log"
	"encoding/base64"

	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

)

var historyBuffer []uint64 // Maintains older history IDs

var messageSet = make(map[string]bool) // To deduplicate messages

// Add a new historyId to the buffer
func addToBuffer(historyID uint64) {
    historyBuffer = append(historyBuffer, historyID)
    if len(historyBuffer) > 1 {
        historyBuffer = historyBuffer[1:]
    }
}

// Get the oldest historyId from the buffer
func getOldestHistoryID() uint64 {
    if len(historyBuffer) == 0 {
        return 0
    }
    return historyBuffer[0]
}

func webhookHandler(c *gin.Context) {
	var pubSubMessage PubSubMessage
	if err := c.ShouldBindJSON(&pubSubMessage); err != nil {
		fmt.Printf("Error parsing Pub/Sub message: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	decodedData, err := pubSubMessage.Message.DecodeData()
  if err != nil {
		fmt.Printf("Error decoding data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}
	fmt.Printf("Decoded data: %+v\n", decodedData)

  err1 := godotenv.Load(".env")
  if err1 != nil {
    fmt.Errorf("Error loading .env file: %v", err)
	}
  accessToken := os.Getenv("ACCESS_TOKEN")
  refreshToken := os.Getenv("REFRESH_TOKEN")
  clientID := os.Getenv("CLIENT_ID")
  clientSecret := os.Getenv("CLIENT_SECRET")
	labelID := os.Getenv("ME_LABEL_ID")

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
		fmt.Errorf("Unable to create Gmail service: %v", err)
	}

	// Inputs
	user := "me"
	historyID := decodedData.HistoryID
  fmt.Println(labelID)

  processHistory(srv, user, historyID)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func processHistory(srv *gmail.Service, user string, historyID uint64) error {
    oldestHistoryID := getOldestHistoryID()
    fmt.Println("in processing history!!")
    fmt.Println(oldestHistoryID)
    if oldestHistoryID == 0 {
        addToBuffer(historyID)
        return fmt.Errorf("no historyId available to query")
    }

    // we don't start with the latest historyid since there will be no changes after that.
    historyCall := srv.Users.History.List(user).StartHistoryId(oldestHistoryID)
    response, err := historyCall.Do()
    if err != nil {
        return fmt.Errorf("error retrieving history: %v", err)
    }

    log.Println(messageSet)
    for _, history := range response.History {
        for _, msgAdded := range history.MessagesAdded {
            if !messageSet[msgAdded.Message.Id] {
                messageSet[msgAdded.Message.Id] = true
                err := processMessage(srv, user, msgAdded.Message.Id)
                if err != nil {
                    log.Printf("Error processing message: %v\n", err)
                }
            }
        }
    }

    // Add the latest historyId to the buffer.
    addToBuffer(historyID)
    return nil
}


func processMessage(srv *gmail.Service, user, messageID string) error {
  messageCall := srv.Users.Messages.Get(user, messageID)
  messageResponse, err := messageCall.Do()
  if err != nil {
    return fmt.Errorf("unable to retrieve message: %v", err)
  }

  if isEmailFrom(messageResponse, os.Getenv("EXPECTED_SENDER")) {
    // Extract the plain text body
    for _, part := range messageResponse.Payload.Parts {
      if part.MimeType == "text/plain" && part.Body.Data != "" {
        body, err := decodeBase64URL(part.Body.Data)
        if err != nil {
          return fmt.Errorf("unable to decode message body: %v", err)
        }
        log.Printf("Message Body: %s\n", body)
      }
    }
  }

  return nil
}

func isEmailFrom(messageResponse *gmail.Message, expectedSender string) bool {
    var fromEmail string

    // Check the "From" field in the message headers
    for _, header := range messageResponse.Payload.Headers {
        if header.Name == "From" {
            fromEmail = header.Value
            break
        }
    }

    if fromEmail == "" {
        log.Println("Message does not have a 'From' field")
        return false
    }

    // Verify if the email is from the specified address
    if !strings.Contains(fromEmail, expectedSender) {
        log.Printf("Message is from %s, skipping\n", fromEmail)
        return false
    }

    return true
}

func decodeBase64URL(data string) (string, error) {
    decodedData, err := base64.URLEncoding.DecodeString(data)
    if err != nil {
        return "", err
    }
    return string(decodedData), nil
}
