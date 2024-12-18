package main

import (
  "github.com/gin-gonic/gin"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/sheets/v4"
)

func setupRoutes(r *gin.Engine, gsrv *gmail.Service, ssrv *sheets.Service, ch chan<- uint64) {
  r.POST("/webhook", webhookHandler(gsrv, ssrv, ch))
}
