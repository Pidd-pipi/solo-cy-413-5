package model

import "time"

// PlanVersion 计划的一次生成版本。新建计划为 v1；每发生一次真实重算递增。
// 被重算替换的版本保留 status=replaced 及其 SnapshotJSON，可在页面回看。
type PlanVersion struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	PlanID  uint   `gorm:"uniqueIndex:uk_plan_version;not null" json:"plan_id"`
	Version int    `gorm:"uniqueIndex:uk_plan_version;not null" json:"version"`
	Status  string `gorm:"size:16;index;not null;default:current" json:"status"`
	// Trigger 触发来源：manual / mood / journal / assessment / init。
	Trigger string `gorm:"size:16;not null;default:init" json:"trigger"`
	// InputHash 生成该版本所依据的近 14 天数据指纹；输入未变则跳过重算，保证同日重复提交幂等。
	InputHash    string     `gorm:"size:64;not null" json:"input_hash"`
	SnapshotJSON string     `gorm:"type:text;not null" json:"snapshot_json"`
	CreatedAt    time.Time  `json:"created_at"`
	ReplacedAt   *time.Time `json:"replaced_at"`
}
