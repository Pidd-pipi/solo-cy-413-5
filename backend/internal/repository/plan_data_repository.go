package repository

import (
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
	"time"
)

// PlanDataRepository 读取近 14 天的情绪 / 日记 / 测评数据，为计划建议引擎提供输入。
// 只读，不修改任何原始记录。
type PlanDataRepository interface {
	// Since 返回 [since, now) 内该用户的情绪记录（按日期升序）。
	MoodsSince(db *gorm.DB, userID uint, since time.Time) ([]model.Mood, error)
	JournalsSince(db *gorm.DB, userID uint, since time.Time) ([]model.Journal, error)
	AssessmentsSince(db *gorm.DB, userID uint, since time.Time) ([]model.UserAssessment, error)
}

type planDataRepository struct{ db *gorm.DB }

func NewPlanDataRepository(db *gorm.DB) PlanDataRepository { return &planDataRepository{db} }

func (r *planDataRepository) MoodsSince(db *gorm.DB, userID uint, since time.Time) (out []model.Mood, e error) {
	e = db.Where("user_id = ? AND record_date >= ?", userID, since).
		Order("record_date asc, id asc").Find(&out).Error
	return
}

func (r *planDataRepository) JournalsSince(db *gorm.DB, userID uint, since time.Time) (out []model.Journal, e error) {
	e = db.Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at asc, id asc").Find(&out).Error
	return
}

func (r *planDataRepository) AssessmentsSince(db *gorm.DB, userID uint, since time.Time) (out []model.UserAssessment, e error) {
	e = db.Where("user_id = ? AND created_at >= ?", userID, since).
		Order("created_at asc, id asc").Find(&out).Error
	return
}
