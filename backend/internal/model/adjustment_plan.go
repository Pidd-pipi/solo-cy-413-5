package model

import "time"

// AdjustmentPlan 七日身心调整计划。
// 每个账号最多同时存在一个进行中（status=active）的计划，
// 该不变量由数据库部分唯一索引 uk_plan_one_active 保证（并发安全）。
type AdjustmentPlan struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"index;not null" json:"user_id"`
	Status    string `gorm:"size:16;index;not null;default:active" json:"status"`
	StartDate time.Time
	EndDate   time.Time
	// FinishedAt 进入 completed/cancelled 等终态的时间，用于历史排序。
	FinishedAt *time.Time
	// ReportJSON 终态时冻结的历史曲线与报告，保证历史报告回读同一结论。
	ReportJSON string    `gorm:"type:text" json:"report_json"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
