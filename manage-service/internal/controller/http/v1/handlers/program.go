package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"manage-service/internal/domain/entity"
	"manage-service/internal/usecase"
)

type ProgramHandler struct {
	uc usecase.Program
}

func NewProgramHandler(uc usecase.Program) *ProgramHandler {
	return &ProgramHandler{uc: uc}
}

// @Summary Список программ
// @Description Получение списка доступных магистерских программ с пагинацией (по умолчанию limit=100)
// @Tags programs
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Количество записей (по умолчанию 100)"
// @Success 200 {array} entity.Program
// @Failure 500 {object} map[string]string
// @Router /v1/programs [get]
func (h *ProgramHandler) ListPrograms(c *gin.Context) {
	programs, err := h.uc.ListPrograms(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, programs)
}

// @Summary Информация о программе
// @Description Получение информации об образовательной программе по ID
// @Tags programs
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "ID программы"
// @Success 200 {object} entity.Program
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/programs/{id} [get]
func (h *ProgramHandler) GetProgram(c *gin.Context) {
	idStr := c.Param("id")
	programID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid program ID"})
		return
	}

	program, err := h.uc.GetProgramByID(c.Request.Context(), programID)
	if err != nil {
		if err.Error() == "program not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Program not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, program)
}

// @Summary Создание программы
// @Description Создаёт новую образовательную программу. Доступно только администратору.
// @Tags programs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body entity.Program true "Данные программы"
// @Success 201 {object} entity.Program
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/programs [post]
func (h *ProgramHandler) CreateProgram(c *gin.Context) {
	var p entity.Program
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.Title == "" || p.Year == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and year are required"})
		return
	}
	created, err := h.uc.CreateProgram(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// @Summary Обновление статуса программы
// @Description Обновляет статус образовательной программы (active/completed). Доступно только администратору.
// @Tags programs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID программы"
// @Param body body map[string]string true "Новый статус"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /v1/programs/{id} [put]
func (h *ProgramHandler) UpdateProgramStatus(c *gin.Context) {
	idStr := c.Param("id")
	programID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program ID"})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	if err := h.uc.UpdateProgramStatus(c.Request.Context(), programID, body.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "program status updated"})
}
