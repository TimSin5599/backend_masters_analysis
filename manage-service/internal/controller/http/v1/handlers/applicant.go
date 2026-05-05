package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"manage-service/internal/usecase"
)

type ApplicantHandler struct {
	uc usecase.Applicant
}

func NewApplicantHandler(uc usecase.Applicant) *ApplicantHandler {
	return &ApplicantHandler{uc: uc}
}
// @Summary     Удаление абитуриента
// @Description Удаляет абитуриента и все связанные с ним данные по ID.
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID абитуриента"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id} [delete]
func (h *ApplicantHandler) DeleteApplicant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	err = h.uc.DeleteApplicant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "applicant deleted successfully"})
}

// @Summary     Удаление данных абитуриента
// @Description Удаляет конкретную запись данных абитуриента (например, запись о работе или достижении) по категории и ID записи.
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       id        path      int     true  "ID абитуриента"
// @Param       category  path      string  true  "Категория данных (work_experience, achievements и т.д.)"
// @Param       dataId    path      int     true  "ID записи"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/data/{category}/{dataId} [delete]
func (h *ApplicantHandler) DeleteApplicantData(c *gin.Context) {
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

	err = h.uc.DeleteApplicantData(c.Request.Context(), applicantID, category, dataID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

// @Summary     Список абитуриентов
// @Description Возвращает список абитуриентов. Можно фильтровать по ID программы.
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       program_id  query     int  false  "ID образовательной программы для фильтрации"
// @Success     200  {array}   entity.Applicant
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants [get]
func (h *ApplicantHandler) ListApplicants(c *gin.Context) {
	programIDStr := c.Query("program_id")
	programID, _ := strconv.ParseInt(programIDStr, 10, 64)

	applicants, err := h.uc.ListApplicants(c.Request.Context(), programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, applicants)
}

type createApplicantRequest struct {
	ProgramID  int64  `json:"program_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Patronymic string `json:"patronymic"`
}

// @Summary     Создание абитуриента
// @Description Создаёт нового абитуриента и привязывает его к образовательной программе.
// @Tags        applicants
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      createApplicantRequest  true  "Данные нового абитуриента"
// @Success     201  {object}  entity.Applicant
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants [post]
func (h *ApplicantHandler) CreateApplicant(c *gin.Context) {
	var request createApplicantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applicant, err := h.uc.CreateApplicant(c.Request.Context(), request.ProgramID, request.FirstName, request.LastName, request.Patronymic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, applicant)
}

// @Summary     Данные абитуриента по категории
// @Description Возвращает структурированные данные абитуриента по указанной категории (identification, education, transcript и т.д.).
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       id        path      int     true   "ID абитуриента"
// @Param       category  query     string  false  "Категория данных"
// @Success     200  {object}  interface{}
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/data [get]
func (h *ApplicantHandler) GetApplicantData(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	category := c.Query("category")
	data, err := h.uc.GetApplicantData(c.Request.Context(), applicantID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// @Summary     Обновление данных абитуриента
// @Description Обновляет данные абитуриента по указанной категории. Принимает произвольный JSON-объект.
// @Tags        applicants
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id        path      int                     true   "ID абитуриента"
// @Param       category  query     string                  false  "Категория данных"
// @Param       body      body      map[string]interface{}  true   "Обновлённые данные"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/data [patch]
func (h *ApplicantHandler) UpdateApplicantData(c *gin.Context) {
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

	err = h.uc.UpdateApplicantData(c.Request.Context(), applicantID, category, rawData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "data updated successfully"})
}

// @Summary     Передача абитуриента оператору
// @Description Переводит абитуриента на этап проверки оператором (изменяет статус заявки).
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       id  path  int  true  "ID абитуриента"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Router      /v1/applicants/{id}/transfer-to-operator [post]
func (h *ApplicantHandler) TransferToOperator(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	err := h.uc.TransferToOperator(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transferred to operator"})
}

// @Summary     Передача абитуриента экспертам
// @Description Переводит абитуриента на этап экспертной оценки (изменяет статус заявки).
// @Tags        applicants
// @Produce     json
// @Security    BearerAuth
// @Param       id  path  int  true  "ID абитуриента"
// @Success     200  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/transfer-to-experts [post]
func (h *ApplicantHandler) TransferToExperts(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	var body struct {
		UserName string `json:"user_name"`
		UserRole string `json:"user_role"`
	}
	_ = c.ShouldBindJSON(&body)

	confirmedBy := ""
	if body.UserName != "" {
		confirmedBy = body.UserName
		if body.UserRole != "" {
			confirmedBy += " (" + body.UserRole + ")"
		}
	}

	err := h.uc.TransferToExperts(c.Request.Context(), applicantID, confirmedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transferred to experts"})
}
