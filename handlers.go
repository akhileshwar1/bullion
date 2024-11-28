package main

import (
	"log"
	"net/http"
	"github.com/gin-gonic/gin"
)

func webhookHandler(c *gin.Context) {
	var pubSubMessage PubSubMessage
	if err := c.ShouldBindJSON(&pubSubMessage); err != nil {
		log.Printf("Error parsing Pub/Sub message: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	decodedData, err := pubSubMessage.Message.DecodeData()
  if err != nil {
		log.Printf("Error decoding data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}
	log.Printf("Decoded data: %+v\n", decodedData)
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
