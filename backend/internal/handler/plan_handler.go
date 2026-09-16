package handler

import (
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/middleware"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"github.com/gin-gonic/gin"
	"log/slog"
	"strconv"
)

func invalidID(entity, field string, err error) *util.AppError {
	return util.NewAppError(constants.CodeValidation, entity+"["+field+"] invalid id", err)
}

type PlanHandler struct {
	s      *service.PlanService
	logger *slog.Logger
}

func NewPlanHandler(s *service.PlanService, l *slog.Logger) *PlanHandler { return &PlanHandler{s, l} }

// Create 幂等：已有进行中计划时返回该计划，不产生重复。
func (h *PlanHandler) Create(c *gin.Context) {
	var r dto.PlanCreateRequest
	if c.Request.ContentLength > 0 {
		if !bind(c, &r) {
			return
		}
	}
	v, isCreated, e := h.s.Create(middleware.UserID(c), r.IdempotencyKey)
	if e != nil {
		c.Error(e)
		return
	}
	if isCreated {
		created(c, v)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Active(c *gin.Context) {
	v, e := h.s.GetActive(middleware.UserID(c))
	if e != nil {
		c.Error(e)
		return
	}
	h.logger.Info(constants.LogPlanListed, "plan_id", v.ID)
	ok(c, v)
}

func (h *PlanHandler) List(c *gin.Context) {
	v, e := h.s.List(middleware.UserID(c))
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Get(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	v, e := h.s.Get(middleware.UserID(c), id)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Regenerate(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	v, e := h.s.Regenerate(middleware.UserID(c), id)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Action(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	var r dto.PlanActionRequest
	if !bind(c, &r) {
		return
	}
	v, e := h.s.Action(middleware.UserID(c), id, r.Action)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Report(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	v, e := h.s.Report(middleware.UserID(c), id)
	if e != nil {
		c.Error(e)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) Version(c *gin.Context) {
	planID, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	versionID, e := strconv.ParseUint(c.Param("versionId"), 10, 64)
	if e != nil {
		c.Error(invalidID("PlanVersion", "id", e))
		return
	}
	v, err := h.s.VersionSnapshot(middleware.UserID(c), planID, uint(versionID))
	if err != nil {
		c.Error(err)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) AddTask(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	var r dto.PlanAddTaskRequest
	if !bind(c, &r) {
		return
	}
	v, e := h.s.AddTask(middleware.UserID(c), id, r)
	if e != nil {
		c.Error(e)
		return
	}
	created(c, v)
}

func (h *PlanHandler) SetTask(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	taskID, e := strconv.ParseUint(c.Param("taskId"), 10, 64)
	if e != nil {
		c.Error(invalidID("PlanTask", "id", e))
		return
	}
	var r dto.PlanTaskRequest
	if !bind(c, &r) {
		return
	}
	v, err := h.s.SetTask(middleware.UserID(c), id, uint(taskID), r)
	if err != nil {
		c.Error(err)
		return
	}
	ok(c, v)
}

func (h *PlanHandler) SetDayNote(c *gin.Context) {
	id, ok2 := parsePlanID(c)
	if !ok2 {
		return
	}
	dayIndex, e := strconv.Atoi(c.Param("dayIndex"))
	if e != nil || dayIndex < 0 || dayIndex >= constants.PlanLen {
		c.Error(invalidID("PlanDay", "day_index", e))
		return
	}
	var r dto.PlanDayNoteRequest
	if !bind(c, &r) {
		return
	}
	v, err := h.s.SetDayNote(middleware.UserID(c), id, dayIndex, r.Note)
	if err != nil {
		c.Error(err)
		return
	}
	ok(c, v)
}

func parsePlanID(c *gin.Context) (uint, bool) {
	id, e := strconv.ParseUint(c.Param("id"), 10, 64)
	if e != nil {
		c.Error(invalidID("Plan", "id", e))
		return 0, false
	}
	return uint(id), true
}
