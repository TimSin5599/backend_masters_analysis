package v1

import (
	"fmt"
	"manage-service/internal/usecase"
	ws "manage-service/internal/websocket"
	"net/http"
	"strconv"
	"time"

	_ "manage-service/internal/entity" // For swagger

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "manage-service/docs"
)

type applicantRoutes struct {
	t     *usecase.ApplicantUseCase
	queue usecase.DocumentQueueRepo
	hub   *ws.Hub
}

func NewRouter(handler *gin.Engine, t *usecase.ApplicantUseCase, queue usecase.DocumentQueueRepo, hub *ws.Hub) {
	// Options
	handler.Use(gin.Recovery())

	// CORS
	handler.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	handler.Use(LoggingMiddleware())

	r := &applicantRoutes{t, queue, hub}

	v1 := handler.Group("/v1")
	{
		v1.GET("/programs", r.listPrograms)
		v1.GET("/programs/:id", r.getProgram)
		v1.GET("/applicants", r.listApplicants)
		v1.POST("/applicants", r.createApplicant)
		v1.DELETE("/applicants/:id", r.deleteApplicant)
		v1.GET("/applicants/:id/data", r.getApplicantData)
		v1.PATCH("/applicants/:id/data", r.updateApplicantData)
		v1.POST("/applicants/:id/documents", r.uploadDocument)
		v1.GET("/applicants/:id/documents/view", r.viewDocument)
		v1.GET("/documents/:id/view", r.viewDocumentByID)
		v1.DELETE("/applicants/:id/data/:category/:dataId", r.deleteApplicantData)
		v1.POST("/documents/:id/reprocess", r.reprocessDocument)
		v1.POST("/applicants/:id/documents/reprocess", r.reprocessLatestDocument)
		v1.GET("/documents/:id/status", r.getDocumentStatus)
		v1.GET("/applicants/:id/queue-status", r.getQueueStatus)
		v1.GET("/applicants/:id/ws", r.websocketHandler)
		
		v1.GET("/applicants/:id/evaluations", r.listExpertEvaluations)
		v1.POST("/applicants/:id/evaluations", r.saveExpertEvaluation)
		v1.GET("/experts/slots", r.getExpertSlots)
		v1.POST("/experts/slots", r.assignExpertSlot)
	}

	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// @Summary Статус документа
// @Description Получаем статус обработки документа
// @Tags documents
// @Accept json
// @Produce json
// @Param id path int true "ID документа"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/documents/{id}/status [get]
func (r *applicantRoutes) getDocumentStatus(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	status, err := r.t.GetDocumentStatus(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// @Summary Повторная обработка последнего документа
// @Description Повторная обработка последнего загруженного документа абитуриента по категории
// @Tags applicants
// @Accept json
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param category query string true "Категория документа"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/documents/reprocess [post]
func (r *applicantRoutes) reprocessLatestDocument(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	docID, err := r.t.ReprocessLatestDocument(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reprocessing started", "document_id": docID})
}

// @Summary Повторная обработка документа
// @Description Запускает повторную обработку конкретного документа
// @Tags documents
// @Accept json
// @Produce json
// @Param id path int true "ID документа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/documents/{id}/reprocess [post]
func (r *applicantRoutes) reprocessDocument(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	err = r.t.ReprocessDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reprocessing started"})
}

// @Summary Удаление абитуриента
// @Description Полностью удаляет абитуриента и связанные с ним данные из системы
// @Tags applicants
// @Accept json
// @Produce json
// @Param id path int true "ID абитуриента"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id} [delete]
func (r *applicantRoutes) deleteApplicant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	err = r.t.DeleteApplicant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "applicant deleted successfully"})
}

// @Summary Удаление данных абитуриента
// @Description Удаляет конкретную запись о данных абитуриенте
// @Tags applicants
// @Accept json
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param category path string true "Категория данных"
// @Param dataId path int true "ID данных"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/data/{category}/{dataId} [delete]
func (r *applicantRoutes) deleteApplicantData(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Param("category")
	dataIdStr := c.Param("dataId")
	dataID, err := strconv.ParseInt(dataIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data id"})
		return
	}

	err = r.t.DeleteApplicantData(c.Request.Context(), applicantID, category, dataID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// @Summary Просмотр документа по ID
// @Description Возвращает файл документа по его ID
// @Tags documents
// @Produce octet-stream
// @Param id path int true "ID документа"
// @Success 200 {file} file
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/documents/{id}/view [get]
func (r *applicantRoutes) viewDocumentByID(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	content, contentType, filename, err := r.t.ViewDocumentByID(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Data(http.StatusOK, contentType, content)
}

// @Summary Просмотр документа абитуриента
// @Description Возвращает последний документ абитуриента по категории
// @Tags applicants
// @Produce octet-stream
// @Param id path int true "ID абитуриента"
// @Param category query string true "Категория документа"
// @Success 200 {file} file
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/applicants/{id}/documents/view [get]
func (r *applicantRoutes) viewDocument(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	content, contentType, filename, err := r.t.ViewDocument(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found or error accessing file"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Data(http.StatusOK, contentType, content)
}

// @Summary Список программ
// @Description Получаем список образовательных программ
// @Tags programs
// @Produce json
// @Success 200 {array} entity.Program
// @Failure 500 {object} map[string]string
// @Router /v1/programs [get]
func (r *applicantRoutes) listPrograms(c *gin.Context) {
	programs, err := r.t.ListPrograms(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, programs)
}

// @Summary Получение программы
// @Description Получение информации о программе
// @Tags programs
// @Produce json
// @Param id path int true "ID программы"
// @Success 200 {object} entity.Program
// @Failure 404 {object} map[string]string
// @Router /v1/programs/{id} [get]
func (r *applicantRoutes) getProgram(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	program, err := r.t.GetProgram(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
		return
	}
	c.JSON(http.StatusOK, program)
}

// @Summary Список абитуриентов
// @Description Получаем список абитуриентов
// @Tags applicants
// @Produce json
// @Param program_id query int false "Фильтр по ID программы"
// @Success 200 {array} entity.Applicant
// @Failure 500 {object} map[string]string
// @Router /v1/applicants [get]
func (r *applicantRoutes) listApplicants(c *gin.Context) {
	programIDStr := c.Query("program_id")
	programID, _ := strconv.ParseInt(programIDStr, 10, 64)

	applicants, err := r.t.ListApplicants(c.Request.Context(), programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, applicants)
}

type createApplicantRequest struct {
	ProgramID  int64  `json:"program_id" binding:"required"`
	FirstName  string `json:"first_name" binding:"required"`
	LastName   string `json:"last_name" binding:"required"`
	Patronymic string `json:"patronymic"`
}

// @Summary Создать абитуриента
// @Description Добавить нового абитуриента
// @Tags applicants
// @Accept json
// @Produce json
// @Param request body createApplicantRequest true "Данные абитуриента"
// @Success 200 {object} entity.Applicant
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants [post]
func (r *applicantRoutes) createApplicant(c *gin.Context) {
	var request createApplicantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applicant, err := r.t.CreateApplicant(c.Request.Context(), request.ProgramID, request.FirstName, request.LastName, request.Patronymic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, applicant)
}

// @Summary Загрузить документ
// @Description Загрузка документа для абитуриента
// @Tags applicants
// @Accept mpfd
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param category formData string true "Категория документа"
// @Param file formData file true "Документ"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/documents [post]
func (r *applicantRoutes) uploadDocument(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.PostForm("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category of the document is required"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Read file content
	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer openedFile.Close()

	content := make([]byte, file.Size)
	_, err = openedFile.Read(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	err = r.t.UploadDocument(c.Request.Context(), applicantID, category, file.Filename, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document uploaded and processing started"})
}
// @Summary Получить данные документа
// @Description Получить данные (парсинг) из документа абитуриента
// @Tags applicants
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param category query string true "Категория документа"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/data [get]
func (r *applicantRoutes) getApplicantData(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Query("category")
	data, err := r.t.GetApplicantData(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// @Summary Обновить данные документа
// @Description Обновить данные абитуриента для категории
// @Tags applicants
// @Accept json
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param category query string true "Категория документа"
// @Param request body map[string]interface{} true "Обновленные данные"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/data [patch]
func (r *applicantRoutes) updateApplicantData(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Query("category")
	var rawData map[string]interface{}
	if err := c.ShouldBindJSON(&rawData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = r.t.UpdateApplicantData(c.Request.Context(), applicantID, category, rawData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "data updated successfully"})
}

// @Summary Получить статус очереди документов
// @Description Возвращает массив статусов очереди для абитуриента
// @Tags applicants
// @Produce json
// @Param id path int true "ID абитуриента"
// @Success 200 {array} entity.DocumentQueueTask
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/applicants/{id}/queue-status [get]
func (r *applicantRoutes) getQueueStatus(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	tasks, err := r.queue.GetByApplicantID(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// @Summary Подключение к WebSocket
// @Description Endpoint для инициализации WebSocket соединения
// @Tags applicants
// @Param id path int true "ID абитуриента"
// @Router /v1/applicants/{id}/ws [get]
func (r *applicantRoutes) websocketHandler(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	r.hub.HandleWebSocket(c, applicantID)
}

type expertEvaluationRequest struct {
	ExpertID int64  `json:"expert_id"`
	Category string `json:"category" binding:"required"`
	Score    int    `json:"score" binding:"required"`
	Comment  string `json:"comment"`
	UserID   int64  `json:"user_id" binding:"required"`
	UserName string `json:"user_name" binding:"required"`
	UserRole string `json:"user_role" binding:"required"`
}

// @Summary Сохранить оценку эксперта
// @Description Сохраняет или обновляет оценку за категорию документа. Лимит 3 эксперта на систему.
// @Tags experts
// @Accept json
// @Produce json
// @Param id path int true "ID абитуриента"
// @Param request body expertEvaluationRequest true "Данные оценки"
// @Success 200 {object} map[string]string
// @Router /v1/applicants/{id}/evaluations [post]
func (r *applicantRoutes) saveExpertEvaluation(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	var req expertEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ExpertID == 0 {
		req.ExpertID = req.UserID
	}

	err := r.t.SaveExpertEvaluation(c.Request.Context(), applicantID, req.ExpertID, req.UserID, req.UserName, req.UserRole, req.Category, req.Score, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "evaluation saved"})
}

// @Summary Список оценок экспертов
// @Description Возвращает все оценки для абитуриента
// @Tags experts
// @Produce json
// @Param id path int true "ID абитуриента"
// @Success 200 {array} entity.ExpertEvaluation
// @Router /v1/applicants/{id}/evaluations [get]
func (r *applicantRoutes) listExpertEvaluations(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	evals, err := r.t.ListExpertEvaluations(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, evals)
}

// @Summary Список слотов экспертов
// @Description Возвращает 3 слота экспертов и назначенных на них пользователей
// @Tags experts
// @Produce json
// @Success 200 {array} entity.ExpertSlot
// @Router /v1/experts/slots [get]
func (r *applicantRoutes) getExpertSlots(c *gin.Context) {
	slots, err := r.t.GetExpertSlots(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slots)
}

type assignSlotRequest struct {
	UserID     int64  `json:"user_id" binding:"required"`
	SlotNumber int    `json:"slot_number" binding:"required"`
	UserRole   string `json:"user_role" binding:"required"`
}

// @Summary Назначить эксперта на слот
// @Description Закрепляет пользователя за слотом Эксперт 1, 2 или 3.
// @Tags experts
// @Accept json
// @Produce json
// @Param request body assignSlotRequest true "Данные слота"
// @Success 200 {object} map[string]string
// @Router /v1/experts/slots [post]
func (r *applicantRoutes) assignExpertSlot(c *gin.Context) {
	var req assignSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := r.t.AssignExpertSlot(c.Request.Context(), req.UserID, req.SlotNumber, req.UserRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "expert slot assigned"})
}

// LoggingMiddleware - middleware для подробного логирования всех HTTP запросов
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		duration := time.Since(startTime)
		fmt.Printf("[MANAGE-SERVICE] %s | %d | %v | %s | %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			c.Writer.Status(),
			duration,
			c.Request.Method,
			c.Request.URL.Path,
		)
	}
}
