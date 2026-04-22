package v1

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "manage-service/docs"
	"manage-service/internal/controller/http/v1/handlers"
	"manage-service/internal/usecase"
	ws "manage-service/internal/websocket"
)

func NewRouter(
	handler *gin.Engine,
	appUC usecase.Applicant,
	docUC usecase.Document,
	expertUC usecase.Expert,
	programUC usecase.Program,
	hub *ws.Hub,
	jwtSecret string,
	corsOrigin string,
) {
	// Options
	handler.Use(gin.Recovery())

	allowOrigin := corsOrigin
	if allowOrigin == "" {
		allowOrigin = "http://localhost:3000"
	}

	handler.Use(cors.New(cors.Config{
		AllowOriginFunc:  isLocalNetworkOrigin,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	handler.Use(LoggingMiddleware())

	appHandler := handlers.NewApplicantHandler(appUC, hub)
	docHandler := handlers.NewDocumentHandler(docUC)
	expertHandler := handlers.NewExpertHandler(expertUC)
	progHandler := handlers.NewProgramHandler(programUC)

	v1 := handler.Group("/v1")
	v1.Use(JWTMiddleware(jwtSecret))
	v1.Use(NoCacheMiddleware())
	{
		v1.GET("/programs", progHandler.ListPrograms)
		v1.POST("/programs", AdminMiddleware(), progHandler.CreateProgram)
		v1.GET("/programs/:id", progHandler.GetProgram)
		v1.PUT("/programs/:id", AdminOrManagerMiddleware(), progHandler.UpdateProgramStatus)

		v1.GET("/applicants", appHandler.ListApplicants)
		v1.POST("/applicants", appHandler.CreateApplicant)
		v1.DELETE("/applicants/:id", appHandler.DeleteApplicant)
		v1.GET("/applicants/:id/data", appHandler.GetApplicantData)
		v1.PATCH("/applicants/:id/data", appHandler.UpdateApplicantData)

		v1.POST("/applicants/:id/documents", docHandler.UploadDocument)
		v1.GET("/applicants/:id/documents", docHandler.ListDocuments)
		v1.GET("/applicants/:id/documents/view", docHandler.ViewDocument)
		v1.DELETE("/applicants/:id/documents/:docId", docHandler.DeleteDocument)
		v1.DELETE("/applicants/:id/data/:category/:dataId", appHandler.DeleteApplicantData)
		v1.POST("/applicants/:id/documents/reprocess", docHandler.ReprocessLatestDocument)
		v1.GET("/applicants/:id/queue-status", docHandler.GetQueueStatus)

		v1.GET("/applicants/:id/ws", appHandler.WebsocketHandler)
		v1.POST("/applicants/:id/transfer-to-operator", appHandler.TransferToOperator)
		v1.POST("/applicants/:id/transfer-to-experts", appHandler.TransferToExperts)

		v1.GET("/applicants/:id/evaluations", expertHandler.ListExpertEvaluations)
		v1.PUT("/applicants/:id/evaluations", expertHandler.SaveExpertEvaluation)
		v1.GET("/applicants/:id/criteria", expertHandler.GetEvaluationCriteria)
		v1.GET("/applicants/:id/scoring-scheme", expertHandler.GetScoringScheme)
		v1.PATCH("/applicants/:id/scoring-scheme", AdminMiddleware(), expertHandler.SetScoringScheme)

		v1.GET("/documents/:id/status", docHandler.GetDocumentStatus)
		v1.PATCH("/documents/:id/status", docHandler.PatchDocumentStatus)
		v1.GET("/documents/:id/view", docHandler.ViewDocumentByID)
		v1.POST("/documents/:id/reprocess", docHandler.ReprocessDocument)
		v1.PATCH("/documents/:id/category", docHandler.ChangeDocumentCategory)

		v1.GET("/experts/slots", expertHandler.GetExpertSlots)
		v1.POST("/experts/slots", expertHandler.AssignExpertSlot)
		v1.GET("/experts", expertHandler.ListExperts)

		v1.GET("/criteria", expertHandler.ListCriteria)
		v1.POST("/criteria", AdminMiddleware(), expertHandler.CreateCriteria)
		v1.PUT("/criteria/:code", AdminMiddleware(), expertHandler.UpdateCriteria)
		v1.DELETE("/criteria/:code", AdminMiddleware(), expertHandler.DeleteCriteria)
	}

	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	fmt.Printf("[MANAGE] Router initialized | CORS origin: %s | JWT protection: enabled\n", allowOrigin)
}
