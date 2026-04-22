package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"statistics-service/internal/usecase"
)

type Handler struct {
	uc *usecase.StatsUseCase
}

func NewRouter(handler *gin.Engine, uc *usecase.StatsUseCase) {
	handler.Use(cors.New(cors.Config{
		AllowOriginFunc:  isLocalNetworkOrigin,
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := &Handler{uc: uc}

	v1 := handler.Group("/v1/stats")
	{
		v1.GET("/overview", h.getOverview)
		v1.GET("/dynamics", h.getDynamics)
	}

	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// getOverview godoc
// @Summary     Общая статистика
// @Description Возвращает сводные метрики по абитуриентам и обработке документов.
// @Description program_id=0 означает все программы.
// @Tags        stats
// @Produce     json
// @Param       program_id  query     int  false  "ID образовательной программы (0 = все)"
// @Success     200  {object}  usecase.GlobalStats
// @Failure     500  {object}  map[string]string
// @Router      /v1/stats/overview [get]
func (h *Handler) getOverview(c *gin.Context) {
	programID, _ := strconv.ParseInt(c.DefaultQuery("program_id", "0"), 10, 64)
	stats, err := h.uc.GetOverview(c.Request.Context(), programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// getDynamics godoc
// @Summary     Динамика обработки
// @Description Возвращает количество абитуриентов по статусам, сгруппированных по дням/неделям/месяцам.
// @Description program_id=0 означает все программы.
// @Tags        stats
// @Produce     json
// @Param       period      query     string  false  "Период агрегации: daily (по умолч.), weekly, monthly"  Enums(daily, weekly, monthly)
// @Param       program_id  query     int     false  "ID образовательной программы (0 = все)"
// @Success     200  {array}   usecase.DailyStats
// @Failure     500  {object}  map[string]string
// @Router      /v1/stats/dynamics [get]
func (h *Handler) getDynamics(c *gin.Context) {
	period := c.DefaultQuery("period", "daily")
	programID, _ := strconv.ParseInt(c.DefaultQuery("program_id", "0"), 10, 64)

	data, err := h.uc.GetDynamics(c.Request.Context(), period, programID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
