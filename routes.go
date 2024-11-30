package main

import (
  "github.com/gin-gonic/gin"
	"google.golang.org/api/gmail/v1"
)

func setupRoutes(r *gin.Engine, srv *gmail.Service, historyBuffer *[]uint64, messageSet map[string]bool) {
	r.POST("/webhook", webhookHandler(srv, historyBuffer, messageSet))
}
