package model

import "time"

// PlanDay 计划中的某一天（day_index: 0-6）。
// 重算只会覆盖“未来且未完成”的天；已完成/已跳过的天及其任务不可被覆盖。
type PlanDay struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	PlanID    uint `gorm:"uniqueIndex:uk_plan_day_index;not null" json:"plan_id"`
	VersionID uint `gorm:"index;not null" json:"version_id"`
	DayIndex  int  `gorm:"uniqueIndex:uk_plan_day_index;not null" json:"day_index"`
	DayDate   time.Time
	Theme     string    `gorm:"size:32;not null" json:"theme"`
	Status    string    `gorm:"size:16;index;not null;default:pending" json:"status"`
	Note      string    `gorm:"type:text" json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
