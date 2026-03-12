package v1

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(handler *gin.Engine) {
	// CORS settings
	handler.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	v1 := handler.Group("/v1/stats")
	{
		v1.GET("/overview", getOverview)
		v1.GET("/dynamics", getDynamics)
	}
}

func getOverview(c *gin.Context) {
	// Mocked for demo
	c.JSON(http.StatusOK, gin.H{
		"total_applicants": 1,
		"processed_today":  1,
		"by_status": gin.H{
			"uploaded":   0,
			"processing": 0,
			"verified":   1,
		},
	})
}

func getDynamics(c *gin.Context) {
	// Mocked for demo
	c.JSON(http.StatusOK, []gin.H{
		{"date": "2024-05-20", "count": 0},
		{"date": "2024-05-21", "count": 1},
	})
}
