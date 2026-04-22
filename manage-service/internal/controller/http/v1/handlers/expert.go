package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"manage-service/internal/domain"
	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"
)

type ExpertHandler struct {
	uc usecase.Expert
}

func NewExpertHandler(uc usecase.Expert) *ExpertHandler {
	return &ExpertHandler{uc: uc}
}

// @Summary     Получение критериев оценки
// @Description Возвращает список всех критериев экспертной оценки. ID абитуриента в пути используется для валидации.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int  true  "ID абитуриента"
// @Success     200  {array}   entity.EvaluationCriteria
// @Failure     400  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/criteria [get]
func (h *ExpertHandler) GetEvaluationCriteria(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}

	criteria, _, err := h.uc.GetEvaluationCriteriaForApplicant(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, criteria)
}

func (h *ExpertHandler) GetScoringScheme(c *gin.Context) {
	applicantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}
	scheme, err := h.uc.GetApplicantScoringScheme(c.Request.Context(), applicantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scheme": scheme})
}

func (h *ExpertHandler) SetScoringScheme(c *gin.Context) {
	applicantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid applicant id"})
		return
	}
	var req setScoringSchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.uc.SetApplicantScoringScheme(c.Request.Context(), applicantID, req.Scheme, req.UserRole); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "scheme updated", "scheme": req.Scheme})
}

// @Summary     Сохранение экспертной оценки
// @Description Сохраняет или обновляет оценки эксперта по критериям для указанного абитуриента. Завершённую оценку изменить нельзя.
// @Tags        experts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path      int                      true  "ID абитуриента"
// @Param       body body      expertEvaluationRequest  true  "Данные оценки"
// @Success     200  {object}  map[string]string
// @Failure     400  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/evaluations [put]
func (h *ExpertHandler) SaveExpertEvaluation(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	var req expertEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ExpertID == "" || req.ExpertID == "0" {
		req.ExpertID = req.UserID
	}

	err := h.uc.SaveExpertEvaluation(c.Request.Context(), applicantID, req.ExpertID, req.UserID, req.UserName, req.UserRole, req.Scores, req.Complete)
	if err != nil {
		status := http.StatusInternalServerError
		if err == domain.ErrEvaluationImmutable {
			status = http.StatusConflict
		} else if err == domain.ErrScoreExceedsMax {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "evaluation saved"})
}

// @Summary     Список оценок экспертов
// @Description Возвращает список экспертных оценок по абитуриенту. Результат фильтруется по роли пользователя: эксперт видит только свои оценки, администратор — все.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Param       id            path      int     true   "ID абитуриента"
// @Param       user_id       query     string  false  "ID текущего пользователя"
// @Param       user_role     query     string  false  "Роль текущего пользователя"
// @Success     200  {array}   entity.ExpertEvaluation
// @Failure     500  {object}  map[string]string
// @Router      /v1/applicants/{id}/evaluations [get]
func (h *ExpertHandler) ListExpertEvaluations(c *gin.Context) {
	idStr := c.Param("id")
	applicantID, _ := strconv.ParseInt(idStr, 10, 64)

	currentUserID := c.Query("user_id")
	if currentUserID == "" {
		currentUserID = c.Query("current_user_id")
	}
	role := c.Query("user_role")

	evals, err := h.uc.ListExpertEvaluations(c.Request.Context(), applicantID, currentUserID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, evals)
}

// @Summary     Список слотов экспертов
// @Description Возвращает все слоты назначения экспертов с их статусами.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   entity.ExpertSlot
// @Failure     500  {object}  map[string]string
// @Router      /v1/experts/slots [get]
func (h *ExpertHandler) GetExpertSlots(c *gin.Context) {
	programIDStr := c.Query("program_id")
	programID, _ := strconv.ParseInt(programIDStr, 10, 64)

	slots, err := h.uc.GetExpertSlots(c.Request.Context(), programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slots)
}

// @Summary     Назначение эксперта на слот
// @Description Назначает пользователя на указанный слот эксперта. Доступно только администратору.
// @Tags        experts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      assignSlotRequest  true  "Данные назначения"
// @Success     200   {object}  map[string]string
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /v1/experts/slots [post]
func (h *ExpertHandler) AssignExpertSlot(c *gin.Context) {
	var req assignSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.uc.AssignExpertSlot(c.Request.Context(), req.UserID, req.SlotNumber, req.ProgramID, req.UserRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "expert slot assigned"})
}

// @Summary     Список экспертов
// @Description Возвращает список всех пользователей с ролью эксперта.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   entity.User
// @Failure     500  {object}  map[string]string
// @Router      /v1/experts [get]
func (h *ExpertHandler) ListExperts(c *gin.Context) {
	experts, err := h.uc.ListExperts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, experts)
}

// @Summary     Список всех критериев оценки
// @Description Возвращает все критерии оценки.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   entity.EvaluationCriteria
// @Failure     500  {object}  map[string]string
// @Router      /v1/criteria [get]
func (h *ExpertHandler) ListCriteria(c *gin.Context) {
	criteria, err := h.uc.GetEvaluationCriteria(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, criteria)
}

// @Summary     Создание критерия оценки
// @Description Создаёт новый критерий оценки. Доступно только администратору.
// @Tags        experts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body      entity.EvaluationCriteria  true  "Данные критерия"
// @Success     201   {object}  map[string]string
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /v1/criteria [post]
func (h *ExpertHandler) CreateCriteria(c *gin.Context) {
	var crit entity.EvaluationCriteria
	if err := c.ShouldBindJSON(&crit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if crit.Code == "" || crit.Title == "" || crit.MaxScore <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code, title and max_score are required"})
		return
	}
	if crit.Type == "" {
		crit.Type = "BASE"
	}
	if crit.Scheme == "" {
		crit.Scheme = "default"
	}
	if err := h.uc.CreateCriteria(c.Request.Context(), crit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "criteria created"})
}

// @Summary     Обновление критерия оценки
// @Description Обновляет существующий критерий оценки по коду. Доступно только администратору.
// @Tags        experts
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       code  path      string                     true  "Код критерия"
// @Param       body  body      entity.EvaluationCriteria  true  "Данные критерия"
// @Success     200   {object}  map[string]string
// @Failure     400   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /v1/criteria/{code} [put]
func (h *ExpertHandler) UpdateCriteria(c *gin.Context) {
	code := c.Param("code")
	var crit entity.EvaluationCriteria
	if err := c.ShouldBindJSON(&crit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	crit.Code = code
	if err := h.uc.UpdateCriteria(c.Request.Context(), crit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "criteria updated"})
}

// @Summary     Удаление критерия оценки
// @Description Удаляет критерий оценки по коду. Доступно только администратору.
// @Tags        experts
// @Produce     json
// @Security    BearerAuth
// @Param       code  path      string  true  "Код критерия"
// @Success     200   {object}  map[string]string
// @Failure     500   {object}  map[string]string
// @Router      /v1/criteria/{code} [delete]
func (h *ExpertHandler) DeleteCriteria(c *gin.Context) {
	code := c.Param("code")
	if err := h.uc.DeleteCriteria(c.Request.Context(), code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "criteria deleted"})
}

type setScoringSchemeRequest struct {
	Scheme   string `json:"scheme"`
	UserRole string `json:"user_role"`
}

type assignSlotRequest struct {
	UserID     string `json:"user_id"`
	SlotNumber int    `json:"slot_number"`
	UserRole   string `json:"role"`
	ProgramID  int64  `json:"program_id"`
}

type expertEvaluationRequest struct {
	ExpertID string                    `json:"expert_id"`
	UserID   string                    `json:"user_id"`
	UserName string                    `json:"user_name"`
	UserRole string                    `json:"user_role"`
	Scores   []entity.ExpertEvaluation `json:"scores"`
	Complete bool                      `json:"complete"`
}
