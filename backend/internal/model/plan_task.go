package model

import "time"

// PlanTask 某一天的具体任务。source 区分系统建议(system)与用户手写(custom)。
// 自定义任务的 slot 以 custom_<uuid> 唯一；系统任务 slot 固定（breathe/move/wind/connect）。
// 重算只删除“未来天”的系统任务，已完成任务与全部自定义任务永远保留。
type PlanTask struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DayID       uint      `gorm:"uniqueIndex:uk_day_slot;not null" json:"day_id"`
	PlanID      uint      `gorm:"index;not null" json:"plan_id"`
	Slot        string    `gorm:"size:48;uniqueIndex:uk_day_slot;not null" json:"slot"`
	Source      string    `gorm:"size:16;not null;default:system" json:"source"`
	Title       string    `gorm:"size:120;not null" json:"title"`
	Content     string    `gorm:"type:text" json:"content"`
	Status      string    `gorm:"size:16;not null;default:pending" json:"status"`
	UserContent string    `gorm:"type:text" json:"user_content"`
	Version     int       `gorm:"not null;default:1" json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
