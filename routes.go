package main

import (
  "github.com/gin-gonic/gin"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/sheets/v4"
)

func setupRoutes(r *gin.Engine, gsrv *gmail.Service, ssrv *sheets.Service, historyBuffer *[]uint64, messageSet map[string]bool) {
	r.POST("/webhook", webhookHandler(gsrv, ssrv, historyBuffer, messageSet))
}
