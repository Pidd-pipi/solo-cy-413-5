package repository

import (
	"errors"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"gorm.io/gorm"
)

// PlanRepository 七日调整计划的写/读仓储。
// 所有写方法接收 *gorm.DB，调用方必须在“按用户咨询锁”开启的事务中执行，保证并发安全。
type PlanRepository interface {
	EnsureIndexes(*gorm.DB) error
	BeginTx() (*gorm.DB, error)
	// AcquireUserLock 获取事务级咨询锁，事务提交/回滚自动释放，串行化同一账号的所有调整操作。
	AcquireUserLock(tx *gorm.DB, userID uint) error

	CreatePlan(tx *gorm.DB, p *model.AdjustmentPlan) error
	UpdatePlan(tx *gorm.DB, p *model.AdjustmentPlan) error
	ActivePlan(db *gorm.DB, userID uint) (*model.AdjustmentPlan, error)
	PlanByID(db *gorm.DB, id, userID uint) (*model.AdjustmentPlan, error)
	ListPlans(db *gorm.DB, userID uint, limit int) ([]model.AdjustmentPlan, error)

	CreateVersion(tx *gorm.DB, v *model.PlanVersion) error
	CurrentVersion(db *gorm.DB, planID uint) (*model.PlanVersion, error)
	VersionByID(db *gorm.DB, id, planID uint) (*model.PlanVersion, error)
	ListVersions(db *gorm.DB, planID uint) ([]model.PlanVersion, error)
	MarkVersionReplaced(tx *gorm.DB, versionID uint) error

	CreateDay(tx *gorm.DB, d *model.PlanDay) error
	UpdateDay(tx *gorm.DB, d *model.PlanDay) error
	DaysByPlan(db *gorm.DB, planID uint) ([]model.PlanDay, error)
	DayByID(db *gorm.DB, id, planID uint) (*model.PlanDay, error)
	// FutureOpenDays 返回 fromIndex 起、状态仍可重算（pending）的天；completed/skipped 不返回。
	FutureOpenDays(tx *gorm.DB, planID uint, fromIndex int) ([]model.PlanDay, error)

	CreateTask(tx *gorm.DB, t *model.PlanTask) error
	UpdateTask(tx *gorm.DB, t *model.PlanTask) error
	DeleteTask(tx *gorm.DB, t *model.PlanTask) error
	TaskByIDForUser(db *gorm.DB, id, userID uint) (*model.PlanTask, error)
	TasksByDay(db *gorm.DB, dayID uint) ([]model.PlanTask, error)
	TasksByPlan(db *gorm.DB, planID uint) ([]model.PlanTask, error)
	// DeletePendingSystemTasks 仅删除指定天中“系统来源且仍待办”的任务；
	// 已完成/已跳过的系统任务与全部自定义（手写）任务永不删除。
	DeletePendingSystemTasks(tx *gorm.DB, dayIDs []uint) error
}

type planRepository struct{ db *gorm.DB }

func NewPlanRepository(db *gorm.DB) PlanRepository { return &planRepository{db} }

func (r *planRepository) EnsureIndexes(db *gorm.DB) error {
	// 部分唯一索引：每个账号最多一个进行中（active/paused）计划。
	// DDL 不支持绑定参数，状态为内部固定可信常量，直接内联。
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_plan_one_active
		ON adjustment_plans(user_id) WHERE status IN ('active','paused')`).Error
}

func (r *planRepository) BeginTx() (*gorm.DB, error) { return r.db.Begin(), nil }

func (r *planRepository) AcquireUserLock(tx *gorm.DB, userID uint) error {
	// 固定命名空间 91413 + userID，作为第二个 int 参数，避免与其他咨询锁键空间冲突。
	return tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", 91413, int64(userID)).Error
}

func (r *planRepository) CreatePlan(tx *gorm.DB, p *model.AdjustmentPlan) error {
	return tx.Create(p).Error
}
func (r *planRepository) UpdatePlan(tx *gorm.DB, p *model.AdjustmentPlan) error {
	return tx.Save(p).Error
}

func (r *planRepository) ActivePlan(db *gorm.DB, userID uint) (*model.AdjustmentPlan, error) {
	var p model.AdjustmentPlan
	e := db.Where("user_id = ? AND status IN ?", userID, constants.PlanActiveStatuses).
		Order("id desc").First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, e
}

func (r *planRepository) PlanByID(db *gorm.DB, id, userID uint) (*model.AdjustmentPlan, error) {
	var p model.AdjustmentPlan
	e := db.Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &p, e
}

func (r *planRepository) ListPlans(db *gorm.DB, userID uint, limit int) (out []model.AdjustmentPlan, e error) {
	if limit <= 0 {
		limit = 20
	}
	e = db.Where("user_id = ?", userID).Order("id desc").Limit(limit).Find(&out).Error
	return
}

func (r *planRepository) CreateVersion(tx *gorm.DB, v *model.PlanVersion) error {
	return tx.Create(v).Error
}

func (r *planRepository) CurrentVersion(db *gorm.DB, planID uint) (*model.PlanVersion, error) {
	var v model.PlanVersion
	e := db.Where("plan_id = ? AND status = ?", planID, constants.PlanVersionCurrent).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *planRepository) VersionByID(db *gorm.DB, id, planID uint) (*model.PlanVersion, error) {
	var v model.PlanVersion
	e := db.Where("id = ? AND plan_id = ?", id, planID).First(&v).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &v, e
}

func (r *planRepository) ListVersions(db *gorm.DB, planID uint) (out []model.PlanVersion, e error) {
	e = db.Where("plan_id = ?", planID).Order("version desc").Find(&out).Error
	return
}

func (r *planRepository) MarkVersionReplaced(tx *gorm.DB, versionID uint) error {
	return tx.Model(&model.PlanVersion{}).Where("id = ?", versionID).
		Update("status", constants.PlanVersionReplaced).Error
}

func (r *planRepository) CreateDay(tx *gorm.DB, d *model.PlanDay) error { return tx.Create(d).Error }
func (r *planRepository) UpdateDay(tx *gorm.DB, d *model.PlanDay) error { return tx.Save(d).Error }

func (r *planRepository) DaysByPlan(db *gorm.DB, planID uint) (out []model.PlanDay, e error) {
	e = db.Where("plan_id = ?", planID).Order("day_index asc").Find(&out).Error
	return
}

func (r *planRepository) DayByID(db *gorm.DB, id, planID uint) (*model.PlanDay, error) {
	var d model.PlanDay
	e := db.Where("id = ? AND plan_id = ?", id, planID).First(&d).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &d, e
}

func (r *planRepository) FutureOpenDays(tx *gorm.DB, planID uint, fromIndex int) (out []model.PlanDay, e error) {
	e = tx.Where("plan_id = ? AND day_index >= ? AND status = ?", planID, fromIndex, constants.PlanItemPending).
		Order("day_index asc").Find(&out).Error
	return
}

func (r *planRepository) CreateTask(tx *gorm.DB, t *model.PlanTask) error { return tx.Create(t).Error }
func (r *planRepository) UpdateTask(tx *gorm.DB, t *model.PlanTask) error { return tx.Save(t).Error }
func (r *planRepository) DeleteTask(tx *gorm.DB, t *model.PlanTask) error { return tx.Delete(t).Error }

func (r *planRepository) TaskByIDForUser(db *gorm.DB, id, userID uint) (*model.PlanTask, error) {
	var t model.PlanTask
	e := db.Joins("JOIN adjustment_plans ON adjustment_plans.id = plan_tasks.plan_id").
		Where("plan_tasks.id = ? AND adjustment_plans.user_id = ?", id, userID).
		First(&t).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &t, e
}

func (r *planRepository) TasksByDay(db *gorm.DB, dayID uint) (out []model.PlanTask, e error) {
	e = db.Where("day_id = ?", dayID).Order("id asc").Find(&out).Error
	return
}

func (r *planRepository) TasksByPlan(db *gorm.DB, planID uint) (out []model.PlanTask, e error) {
	e = db.Where("plan_id = ?", planID).Order("id asc").Find(&out).Error
	return
}

func (r *planRepository) DeletePendingSystemTasks(tx *gorm.DB, dayIDs []uint) error {
	if len(dayIDs) == 0 {
		return nil
	}
	return tx.Where("day_id IN ? AND source = ? AND status = ?", dayIDs, constants.PlanTaskSourceSystem, constants.PlanItemPending).
		Delete(&model.PlanTask{}).Error
}
