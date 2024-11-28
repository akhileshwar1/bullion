package main

import "github.com/gin-gonic/gin"

func setupRoutes(r *gin.Engine) {
	r.POST("/webhook", webhookHandler)
}
