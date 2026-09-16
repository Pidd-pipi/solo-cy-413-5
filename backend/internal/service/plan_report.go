package service

import (
	"encoding/json"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"time"
)

// BuildReport 仅依据“不可变的完成记录”（天状态与任务完成状态）计算历史曲线。
// 已完成任务与天不会被重算覆盖，因此任意时刻（暂停/恢复/结束/取消后）回读都得到同一结论。
func BuildReport(p *model.AdjustmentPlan, days []model.PlanDay, tasks []model.PlanTask) dto.PlanReport {
	byDay := map[uint][]model.PlanTask{}
	for _, t := range tasks {
		byDay[t.DayID] = append(byDay[t.DayID], t)
	}
	rep := dto.PlanReport{
		PlanID:     p.ID,
		Status:     p.Status,
		StartDate:  p.StartDate.Format("2006-01-02"),
		EndDate:    p.EndDate.Format("2006-01-02"),
		TotalTasks: 0,
		DoneTasks:  0,
		Days:       []dto.PlanReportDay{},
	}
	if p.FinishedAt != nil {
		v := p.FinishedAt.Format(time.RFC3339)
		rep.FinishedAt = &v
	}
	for _, d := range days {
		ts := byDay[d.ID]
		done := 0
		for _, t := range ts {
			if t.Status == constants.PlanItemDone {
				done++
			}
		}
		total := len(ts)
		rate := 0.0
		if total > 0 {
			rate = float64(done) / float64(total)
		}
		rep.TotalTasks += total
		rep.DoneTasks += done
		rep.Days = append(rep.Days, dto.PlanReportDay{
			DayIndex:       d.DayIndex,
			Date:           d.DayDate.Format("2006-01-02"),
			Status:         d.Status,
			TotalTasks:     total,
			DoneTasks:      done,
			CompletionRate: rate,
		})
	}
	if rep.TotalTasks > 0 {
		rep.CompletionRate = float64(rep.DoneTasks) / float64(rep.TotalTasks)
	}
	rep.Summary = reportSummary(p.Status, rep)
	return rep
}

func reportSummary(status string, rep dto.PlanReport) []string {
	out := []string{}
	switch status {
	case constants.PlanStatusCompleted:
		out = append(out, "计划已结束，这份曲线来自你每天真实完成的记录。")
	case constants.PlanStatusCancelled:
		out = append(out, "计划已取消，已完成的天数与任务被原样保留，不会再变化。")
	case constants.PlanStatusPaused:
		out = append(out, "计划已暂停，历史完成情况保持不变，恢复后可继续。")
	default:
		out = append(out, "计划进行中，曲线随你的完成情况更新，已完成的部分不会被重算覆盖。")
	}
	return out
}

func marshalReport(rep dto.PlanReport) string {
	b, _ := json.Marshal(rep)
	return string(b)
}
