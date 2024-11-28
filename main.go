package main

import (
	"log"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	setupRoutes(r)
	if err := r.Run("0.0.0.0:3000"); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}
