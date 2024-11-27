package main

import (
	"log"
	"net/http"
	"github.com/gin-gonic/gin"
)

type PubSubMessage struct {
	Message struct {
		Data       string `json:"data"`
		MessageID  string `json:"messageId"`
		Attributes struct {
			HistoryID string `json:"historyId"`
		} `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

func main() {
  r := gin.Default()

  // Define the webhook endpoint
  r.POST("/webhook", func(c *gin.Context) {
    var pubSubMessage PubSubMessage

    // Parse the JSON body into the PubSubMessage struct
    if err := c.ShouldBindJSON(&pubSubMessage); err != nil {
      log.Printf("Error parsing Pub/Sub message: %v\n", err)
      c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
      return
    }

    // Decode the history ID and log it
    historyID := pubSubMessage.Message.Attributes.HistoryID
    log.Printf("Received history ID: %s\n", historyID)

    // Respond with 200 OK
    c.JSON(http.StatusOK, gin.H{"status": "success"})
  })

  // Start the server
  if err := r.Run("127.0.0.1:8080"); err != nil {
    log.Fatalf("Error starting server: %v\n", err)
  }
}
