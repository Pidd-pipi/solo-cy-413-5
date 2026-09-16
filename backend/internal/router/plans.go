package router

import (
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

// RegisterPlans 注册七日身心调整计划路由。全部需要登录（携带 JWT）。
func RegisterPlans(g *gin.RouterGroup, h *handler.PlanHandler, auth gin.HandlerFunc) {
	p := g.Group("/plans", auth)
	p.POST("", h.Create)                            // 生成/幂等获取进行中计划
	p.GET("/active", h.Active)                      // 当前进行中计划
	p.GET("", h.List)                               // 历史计划列表
	p.GET("/:id", h.Get)                            // 计划详情（含来源/每日/版本）
	p.POST("/:id/regenerate", h.Regenerate)         // 手动重算后续建议
	p.POST("/:id/actions", h.Action)                // 暂停/恢复/结束/取消
	p.GET("/:id/report", h.Report)                  // 历史曲线与报告（终态冻结）
	p.GET("/:id/versions/:versionId", h.Version)    // 回看某版本（含被替换版本）
	p.POST("/:id/tasks", h.AddTask)                 // 追加手写任务
	p.PUT("/:id/tasks/:taskId", h.SetTask)          // 勾选任务/补充手写感受
	p.PUT("/:id/days/:dayIndex/note", h.SetDayNote) // 保存某天手写备注
}
