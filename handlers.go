package main

import (
	"io"
	"log"
	"net/http"
	"github.com/gin-gonic/gin"
)

type PubSubMessage struct {
	Message struct {
		Attributes struct {
			HistoryID string `json:"history_id"`
		} `json:"attributes"`
	} `json:"message"`
}

func webhookHandler(c *gin.Context) {
	var pubSubMessage PubSubMessage

	if err := c.ShouldBindJSON(&pubSubMessage); err != nil {
		log.Printf("Error parsing Pub/Sub message: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	historyID := pubSubMessage.Message.Attributes.HistoryID
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	rawJSON := string(bodyBytes)
	log.Printf("Received history ID in message: %s, %s\n", historyID, rawJSON)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
