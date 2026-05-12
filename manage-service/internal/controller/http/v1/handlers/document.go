package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"manage-service/internal/usecase"
	"manage-service/pkg/metrics"
)

type DocumentHandler struct {
	uc usecase.Document
}

func NewDocumentHandler(uc usecase.Document) *DocumentHandler {
	return &DocumentHandler{uc: uc}
}

// @Summary     Статус документа
// @Description Возвращает текущий статус обработки документа по его ID.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID документа"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Router      /v1/documents/{id}/status [get]
func (h *DocumentHandler) GetDocumentStatus(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	status, err := h.uc.GetDocumentStatus(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// @Summary     Обновление статуса документа
// @Description Обновляет статус документа вручную. Разрешён только переход в "completed" (после ручного ввода данных оператором).
// @Tags        documents
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id    path      int                     true  "ID документа"
// @Param       body  body      patchStatusRequest      true  "Новый статус"
// @Success     200   {object}  map[string]string
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /v1/documents/{id}/status [patch]
func (h *DocumentHandler) PatchDocumentStatus(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var body patchStatusRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	// Разрешаем только переход в "completed" через этот эндпоинт
	allowed := map[string]bool{"completed": true}
	if !allowed[body.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("status transition to '%s' is not allowed via this endpoint", body.Status)})
		return
	}

	if err := h.uc.UpdateDocumentStatus(c.Request.Context(), docID, body.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

type patchStatusRequest struct {
	Status string `json:"status"`
}

// @Summary     Повторная обработка последнего документа
// @Description Запускает повторное AI-извлечение для последнего загруженного документа абитуриента по указанной категории.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id        path      int     true  "ID абитуриента"
// @Param       category  query     string  true  "Категория документа"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/documents/reprocess [post]
func (h *DocumentHandler) ReprocessLatestDocument(c *gin.Context) {
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

	docID, err := h.uc.ReprocessLatestDocument(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reprocessing started", "document_id": docID})
}

// @Summary     Повторная обработка документа по ID
// @Description Запускает повторное AI-извлечение для конкретного документа по его ID.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID документа"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/documents/{id}/reprocess [post]
func (h *DocumentHandler) ReprocessDocument(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	err = h.uc.ReprocessDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reprocessing started"})
}

// @Summary     Удаление документа
// @Description Удаляет документ и все связанные с ним извлечённые данные по ID абитуриента и ID документа.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id     path      int  true  "ID абитуриента"
// @Param       docId  path      int  true  "ID документа"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/documents/{docId} [delete]
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	docIdStr := c.Param("docId")
	docID, err := strconv.ParseInt(docIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	err = h.uc.DeleteDocument(c.Request.Context(), applicantID, docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document and associated data deleted successfully"})
}

// @Summary     Просмотр документа по ID
// @Description Возвращает содержимое файла документа напрямую (inline) по ID документа. Content-Type определяется типом файла.
// @Tags        documents
// @Produce     application/octet-stream
// @Security    BearerAuth
// @Param       id   path      int  true  "ID документа"
// @Success     200  {file}    binary
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Router      /v1/documents/{id}/view [get]
func (h *DocumentHandler) ViewDocumentByID(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	content, contentType, filename, err := h.uc.ViewDocumentByID(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Data(http.StatusOK, contentType, content)
}

// @Summary     Просмотр последнего документа абитуриента
// @Description Возвращает содержимое последнего загруженного файла по ID абитуриента и категории. Content-Type определяется типом файла.
// @Tags        documents
// @Produce     application/octet-stream
// @Security    BearerAuth
// @Param       id        path      int     true  "ID абитуриента"
// @Param       category  query     string  true  "Категория документа"
// @Success     200  {file}    binary
// @Failure     400  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Router      /v1/applicants/{id}/documents/view [get]
func (h *DocumentHandler) ViewDocument(c *gin.Context) {
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

	content, contentType, filename, err := h.uc.ViewDocument(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found or error accessing file"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Data(http.StatusOK, contentType, content)
}

// @Summary     Загрузка документа
// @Description Загружает файл документа для абитуриента, сохраняет в MinIO и запускает очередь AI-обработки.
// @Tags        documents
// @Accept      multipart/form-data
// @Produce     json
// @Security    BearerAuth
// @Param       id        path      int     true   "ID абитуриента"
// @Param       category  formData  string  true   "Категория документа"
// @Param       doc_type  formData  string  false  "Тип документа (например, для профессиональной переподготовки)"
// @Param       file      formData  file    true   "Файл документа (PDF)"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/documents [post]
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.PostForm("category")
	if category == "" {
		category = "unknown"
	}
	docType := c.PostForm("doc_type")
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

	doc, err := h.uc.UploadDocument(c.Request.Context(), applicantID, category, file.Filename, content, docType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.DocumentsUploadedTotal.WithLabelValues(category).Inc()
	c.JSON(http.StatusOK, gin.H{"message": "document uploaded and processing started", "document_id": doc.ID})
}

// @Summary     Список документов абитуриента
// @Description Возвращает список всех загруженных документов абитуриента с их метаданными и статусами обработки.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID абитуриента"
// @Success     200  {array}   entity.Document
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/documents [get]
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	docs, err := h.uc.GetDocuments(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// @Summary     Статус очереди обработки
// @Description Возвращает список задач в очереди AI-обработки документов для указанного абитуриента.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID абитуриента"
// @Success     200  {array}   entity.DocumentQueueTask
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/queue-status [get]
func (h *DocumentHandler) GetQueueStatus(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	tasks, err := h.uc.GetQueueTasks(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// (Staging endpoints removed)

// @Summary     Изменение категории документа
// @Description Меняет категорию документа на новую и запускает его повторную обработку.
// @Tags        documents
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID документа"
// @Param       category body map[string]string true "Новая категория: { \"category\": \"new_category\" }"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/documents/{id}/category [patch]
func (h *DocumentHandler) ChangeDocumentCategory(c *gin.Context) {
	idStr := c.Param("id")
	docID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	category, ok := req["category"]
	if !ok || category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	err = h.uc.ChangeDocumentCategory(c.Request.Context(), docID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document category changed and reprocessing started"})
}
