package util

import (
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"time"
)

func FormatDate(v time.Time) string { return v.Format("2006-01-02") }
func MoodText(tag string) string {
	m := map[string]string{constants.MoodHappy: "开心", constants.MoodAnxious: "焦虑", constants.MoodTired: "疲惫", constants.MoodAngry: "愤怒", constants.MoodCalm: "平静"}
	return m[tag]
}
func AssessmentText(c string) string {
	m := map[string]string{constants.AssessmentAnxiety: "焦虑", constants.AssessmentDepression: "抑郁", constants.AssessmentStress: "压力", constants.AssessmentSleep: "睡眠"}
	return m[c]
}
func ThemeColor(t string) string { return constants.ThemeColors[t] }

func PlanStatusText(s string) string {
	m := map[string]string{
		constants.PlanStatusActive:    "进行中",
		constants.PlanStatusPaused:    "已暂停",
		constants.PlanStatusCompleted: "已结束",
		constants.PlanStatusCancelled: "已取消",
	}
	return m[s]
}

func PlanTriggerText(t string) string {
	m := map[string]string{
		constants.PlanTriggerInit:       "初次生成",
		constants.PlanTriggerManual:     "手动重算",
		constants.PlanTriggerMood:       "新情绪记录",
		constants.PlanTriggerJournal:    "新日记",
		constants.PlanTriggerAssessment: "新测评结果",
	}
	return m[t]
}

func PlanTaskSourceText(s string) string {
	if s == constants.PlanTaskSourceCustom {
		return "手写"
	}
	return "建议"
}
