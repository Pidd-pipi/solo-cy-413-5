package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// PlanService 编排七日计划的生成、重算、生命周期与历史报告。
// 所有写操作都在“按用户咨询锁 + 事务”内执行，串行化同一账号，杜绝并发产生重复任务或串档。
type PlanService struct {
	db     *gorm.DB
	plans  repository.PlanRepository
	data   repository.PlanDataRepository
	logger *slog.Logger
}

func NewPlanService(db *gorm.DB, p repository.PlanRepository, d repository.PlanDataRepository, l *slog.Logger) *PlanService {
	return &PlanService{db: db, plans: p, data: d, logger: l}
}

// snapshotFile 是每个版本冻结的建议快照（来源 + 七天系统建议），用于回看被替换版本。
type snapshotFile struct {
	Source dto.PlanSourceView `json:"source"`
	Days   []snapshotDay      `json:"days"`
}
type snapshotDay struct {
	DayIndex int       `json:"day_index"`
	Theme    string    `json:"theme"`
	Tasks    []RecTask `json:"tasks"`
}

// inLock 在事务内持有该账号的咨询锁执行 fn。
func (s *PlanService) inLock(uid uint, fn func(tx *gorm.DB) error) error {
	tx, e := s.plans.BeginTx()
	if e != nil {
		return fmt.Errorf("Plan[user_id=%d] begin tx failed: %w", uid, e)
	}
	if e = s.plans.AcquireUserLock(tx, uid); e != nil {
		tx.Rollback()
		return fmt.Errorf("Plan[user_id=%d] lock failed: %w", uid, e)
	}
	if e = fn(tx); e != nil {
		tx.Rollback()
		return e
	}
	if e = tx.Commit().Error; e != nil {
		return fmt.Errorf("Plan[user_id=%d] commit failed: %w", uid, e)
	}
	return nil
}

// ---- 创建 ----

// Create 幂等：若已有进行中计划，直接返回它（不新建、不产生重复任务）。
func (s *PlanService) Create(uid uint, idemKey string) (view *dto.PlanView, created bool, e error) {
	e = s.inLock(uid, func(tx *gorm.DB) error {
		if existing, err := s.plans.ActivePlan(tx, uid); err == nil {
			v, err := s.buildView(tx, existing)
			if err != nil {
				return err
			}
			view, created = v, false
			s.logger.Info(constants.LogPlanCreatedDedup, "plan_id", existing.ID, "idempotency_key", idemKey)
			return nil
		} else if !errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("Plan[user_id=%d] active lookup failed: %w", uid, err)
		}

		start := truncateDay(time.Now())
		sig, err := s.loadSignal(tx, uid)
		if err != nil {
			return err
		}
		p := &model.AdjustmentPlan{UserID: uid, Status: constants.PlanStatusActive, StartDate: start, EndDate: start.AddDate(0, 0, constants.PlanLen-1)}
		if err = s.plans.CreatePlan(tx, p); err != nil {
			return fmt.Errorf("Plan[user_id=%d] create failed: %w", uid, err)
		}
		recs := GeneratePlan(sig)
		if err = s.insertVersionAndDays(tx, p, 1, constants.PlanTriggerInit, sig, start, recs); err != nil {
			return err
		}
		v, err := s.buildView(tx, p)
		if err != nil {
			return err
		}
		view, created = v, true
		s.logger.Info(constants.LogPlanCreated, "plan_id", p.ID, "user_id", uid, "input_hash", sig.Fingerprint)
		return nil
	})
	return
}

func (s *PlanService) loadSignal(db *gorm.DB, uid uint) (PlanSignal, error) {
	since := truncateDay(time.Now()).AddDate(0, 0, -(planWindowDays - 1))
	moods, e := s.data.MoodsSince(db, uid, since)
	if e != nil {
		return PlanSignal{}, fmt.Errorf("Plan[user_id=%d] load moods failed: %w", uid, e)
	}
	journals, e := s.data.JournalsSince(db, uid, since)
	if e != nil {
		return PlanSignal{}, fmt.Errorf("Plan[user_id=%d] load journals failed: %w", uid, e)
	}
	assess, e := s.data.AssessmentsSince(db, uid, since)
	if e != nil {
		return PlanSignal{}, fmt.Errorf("Plan[user_id=%d] load assessments failed: %w", uid, e)
	}
	return BuildSignal(moods, journals, assess), nil
}

func (s *PlanService) insertVersionAndDays(tx *gorm.DB, p *model.AdjustmentPlan, no int, trigger string, sig PlanSignal, start time.Time, recs []RecDay) error {
	snap := snapshotFile{Source: toSource(sig), Days: make([]snapshotDay, 0, len(recs))}
	ver := &model.PlanVersion{PlanID: p.ID, Version: no, Status: constants.PlanVersionCurrent, Trigger: trigger, InputHash: sig.Fingerprint, SnapshotJSON: ""}
	for _, d := range recs {
		tasks := make([]RecTask, 0, len(d.Tasks))
		tasks = append(tasks, d.Tasks...)
		snap.Days = append(snap.Days, snapshotDay{DayIndex: d.DayIndex, Theme: d.Theme, Tasks: tasks})
	}
	b, _ := json.Marshal(snap)
	ver.SnapshotJSON = string(b)
	if e := s.plans.CreateVersion(tx, ver); e != nil {
		return fmt.Errorf("PlanVersion[plan_id=%d] create failed: %w", p.ID, e)
	}
	for _, d := range recs {
		day := &model.PlanDay{PlanID: p.ID, VersionID: ver.ID, DayIndex: d.DayIndex, DayDate: start.AddDate(0, 0, d.DayIndex), Theme: d.Theme, Status: constants.PlanItemPending}
		if e := s.plans.CreateDay(tx, day); e != nil {
			return fmt.Errorf("PlanDay[plan_id=%d] create failed: %w", p.ID, e)
		}
		for _, t := range d.Tasks {
			if e := s.plans.CreateTask(tx, &model.PlanTask{DayID: day.ID, PlanID: p.ID, Slot: t.Slot, Source: constants.PlanTaskSourceSystem, Title: t.Title, Content: t.Content, Status: constants.PlanItemPending, Version: no}); e != nil {
				return fmt.Errorf("PlanTask[day_id=%d] create failed: %w", day.ID, e)
			}
		}
	}
	return nil
}

// ---- 重算 ----

// recomputeFuture 在锁内用最新信号重算“今天及以后、仍 pending”的天。
// 已完成/已跳过的天、全部自定义任务、所有手写内容均不触碰。
// 只要去重后的有效输入指纹未变（含同日重复提交的情形），一律不产生新版本、不产生重复任务——
// 即使是用户手动触发重算也一样。
func (s *PlanService) recomputeFuture(tx *gorm.DB, p *model.AdjustmentPlan, trigger string) (bool, error) {
	cur, e := s.plans.CurrentVersion(tx, p.ID)
	if e != nil {
		return false, fmt.Errorf("PlanVersion[plan_id=%d] current lookup failed: %w", p.ID, e)
	}
	sig, e := s.loadSignal(tx, p.UserID)
	if e != nil {
		return false, e
	}
	if sig.Fingerprint == cur.InputHash {
		return false, nil
	}
	from := todayIndex(p.StartDate)
	if from < 0 {
		from = 0
	}
	openDays, e := s.plans.FutureOpenDays(tx, p.ID, from)
	if e != nil {
		return false, fmt.Errorf("PlanDay[plan_id=%d] future lookup failed: %w", p.ID, e)
	}
	if len(openDays) == 0 {
		return false, nil
	}
	recs := GeneratePlan(sig)
	recByIndex := map[int]RecDay{}
	for _, r := range recs {
		recByIndex[r.DayIndex] = r
	}

	newNo := cur.Version + 1
	now := time.Now()
	if e = s.plans.MarkVersionReplaced(tx, cur.ID); e != nil {
		return false, fmt.Errorf("PlanVersion[id=%d] replace failed: %w", cur.ID, e)
	}
	// 显式写 ReplacedAt（MarkVersionReplaced 只更新 status）。
	if e = tx.Model(&model.PlanVersion{}).Where("id = ?", cur.ID).Update("replaced_at", now).Error; e != nil {
		return false, e
	}

	snap := snapshotFile{Source: toSource(sig), Days: []snapshotDay{}}
	for _, r := range recs {
		snap.Days = append(snap.Days, snapshotDay{DayIndex: r.DayIndex, Theme: r.Theme, Tasks: r.Tasks})
	}
	b, _ := json.Marshal(snap)
	nv := &model.PlanVersion{PlanID: p.ID, Version: newNo, Status: constants.PlanVersionCurrent, Trigger: trigger, InputHash: sig.Fingerprint, SnapshotJSON: string(b)}
	if e = s.plans.CreateVersion(tx, nv); e != nil {
		return false, fmt.Errorf("PlanVersion[plan_id=%d] create failed: %w", p.ID, e)
	}

	dayIDs := make([]uint, 0, len(openDays))
	for _, d := range openDays {
		dayIDs = append(dayIDs, d.ID)
	}
	// 仅删除这些未来天上“系统来源且仍待办”的任务；
	// 已完成/已跳过的系统任务、以及全部手写任务都保留（绝不覆盖已完成与手写内容）。
	if e = s.plans.DeletePendingSystemTasks(tx, dayIDs); e != nil {
		return false, fmt.Errorf("PlanTask[plan_id=%d] prune failed: %w", p.ID, e)
	}
	for _, d := range openDays {
		r, ok := recByIndex[d.DayIndex]
		if !ok {
			continue
		}
		d.Theme = r.Theme
		d.VersionID = nv.ID
		if e = s.plans.UpdateDay(tx, &d); e != nil {
			return false, fmt.Errorf("PlanDay[id=%d] update failed: %w", d.ID, e)
		}
		// 计算该天仍被占用的槽位（已保留的系统任务 + 手写任务），新建议跳过这些槽位避免唯一约束冲突与覆盖。
		kept, err := s.plans.TasksByDay(tx, d.ID)
		if err != nil {
			return false, fmt.Errorf("PlanTask[day_id=%d] kept lookup failed: %w", d.ID, err)
		}
		occupied := map[string]bool{}
		for _, k := range kept {
			occupied[k.Slot] = true
		}
		for _, t := range r.Tasks {
			if occupied[t.Slot] {
				continue
			}
			if e = s.plans.CreateTask(tx, &model.PlanTask{DayID: d.ID, PlanID: p.ID, Slot: t.Slot, Source: constants.PlanTaskSourceSystem, Title: t.Title, Content: t.Content, Status: constants.PlanItemPending, Version: newNo}); e != nil {
				return false, fmt.Errorf("PlanTask[day_id=%d] create failed: %w", d.ID, e)
			}
			occupied[t.Slot] = true
		}
	}
	s.logger.Info(constants.LogPlanRecomputed, "plan_id", p.ID, "version", newNo, "trigger", trigger, "days", len(openDays), "input_hash", sig.Fingerprint)
	return true, nil
}

// OnSourceDataChanged 供情绪/日记/测评写入后回调；进行中计划才自动重算，错误不阻断原写入。
func (s *PlanService) OnSourceDataChanged(uid uint, trigger string) {
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.plans.ActivePlan(tx, uid)
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if p.Status != constants.PlanStatusActive {
			return nil
		}
		_, err = s.recomputeFuture(tx, p, trigger)
		return err
	})
	if e != nil {
		s.logger.Error(constants.LogPlanRecomputeSkipped, "user_id", uid, "trigger", trigger, "error", e)
	}
}

// Regenerate 用户手动重算（仅进行中）。
func (s *PlanService) Regenerate(uid, planID uint) (*dto.PlanView, error) {
	view := &dto.PlanView{}
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.requireActivePlan(tx, uid, planID)
		if err != nil {
			return err
		}
		if _, err = s.recomputeFuture(tx, p, constants.PlanTriggerManual); err != nil {
			return err
		}
		p, err = s.plans.PlanByID(tx, p.ID, uid)
		if err != nil {
			return err
		}
		view, err = s.buildView(tx, p)
		return err
	})
	return view, e
}

// ---- 生命周期 ----

func (s *PlanService) Action(uid, planID uint, action string) (*dto.PlanView, error) {
	view := &dto.PlanView{}
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.plans.PlanByID(tx, planID, uid)
		if err != nil {
			return fmt.Errorf("Plan[id=%d] fetch failed: %w", planID, err)
		}
		switch action {
		case "pause":
			if p.Status != constants.PlanStatusActive {
				return util.NewAppError(constants.CodeConflict, fmt.Sprintf("Plan[id=%d] pause failed: not active", planID), nil)
			}
			p.Status = constants.PlanStatusPaused
		case "resume":
			if p.Status != constants.PlanStatusPaused {
				return util.NewAppError(constants.CodeConflict, fmt.Sprintf("Plan[id=%d] resume failed: not paused", planID), nil)
			}
			p.Status = constants.PlanStatusActive
		case "complete", "cancel":
			if p.Status != constants.PlanStatusActive && p.Status != constants.PlanStatusPaused {
				return util.NewAppError(constants.CodeConflict, fmt.Sprintf("Plan[id=%d] %s failed: already terminal", planID, action), nil)
			}
			if action == "complete" {
				p.Status = constants.PlanStatusCompleted
			} else {
				p.Status = constants.PlanStatusCancelled
			}
			fin := time.Now()
			p.FinishedAt = &fin
		default:
			return util.NewAppError(constants.CodeValidation, fmt.Sprintf("Plan[id=%d] action failed: %s", planID, action), nil)
		}

		// 恢复时用暂停期间的新数据补一次重算（仅未来天，幂等）。
		if action == "resume" {
			if _, err = s.recomputeFuture(tx, p, constants.PlanTriggerManual); err != nil {
				return err
			}
		}
		// 进入终态：冻结报告，保证历史曲线/结论此后恒定。
		if action == "complete" || action == "cancel" {
			days, tks, err := s.loadDaysTasks(tx, p.ID)
			if err != nil {
				return err
			}
			rep := BuildReport(p, days, tks)
			p.ReportJSON = marshalReport(rep)
		}
		if err = s.plans.UpdatePlan(tx, p); err != nil {
			return fmt.Errorf("Plan[id=%d] status update failed: %w", planID, err)
		}
		s.logger.Info(constants.LogPlanStatusChanged, "plan_id", planID, "action", action, "status", p.Status)
		view, err = s.buildView(tx, p)
		return err
	})
	return view, e
}

// ---- 任务 / 手写内容 ----

func (s *PlanService) SetTask(uid, planID, taskID uint, req dto.PlanTaskRequest) (*dto.PlanTaskView, error) {
	out := &dto.PlanTaskView{}
	e := s.inLock(uid, func(tx *gorm.DB) error {
		t, err := s.plans.TaskByIDForUser(tx, taskID, uid)
		if err != nil {
			return fmt.Errorf("PlanTask[id=%d] fetch failed: %w", taskID, err)
		}
		if t.PlanID != planID {
			return util.NewAppError(constants.CodeNotFound, fmt.Sprintf("PlanTask[id=%d] not belong to Plan[id=%d]", taskID, planID), nil)
		}
		p, err := s.plans.PlanByID(tx, t.PlanID, uid)
		if err != nil {
			return fmt.Errorf("Plan[id=%d] fetch failed: %w", t.PlanID, err)
		}
		if p.Status != constants.PlanStatusActive {
			return util.NewAppError(constants.CodeConflict, fmt.Sprintf("Plan[id=%d] task update failed: plan not active", p.ID), nil)
		}
		if req.Status != "" {
			t.Status = req.Status
			t.Version++
		}
		if req.UserContent != "" || req.Status != "" {
			// 手写感受一旦写入即保留；重算不会清空。显式传空字符串且仅改状态时不覆盖。
			if req.UserContent != "" {
				t.UserContent = req.UserContent
			}
		}
		if err = s.plans.UpdateTask(tx, t); err != nil {
			return fmt.Errorf("PlanTask[id=%d] update failed: %w", taskID, err)
		}

		if err = s.refreshDayAndMaybeComplete(tx, p, t.DayID); err != nil {
			return err
		}
		*out = toTaskView(*t)
		s.logger.Info(constants.LogPlanTaskChanged, "plan_task_id", taskID, "status", t.Status)
		return nil
	})
	return out, e
}

func (s *PlanService) AddTask(uid, planID uint, req dto.PlanAddTaskRequest) (*dto.PlanTaskView, error) {
	out := &dto.PlanTaskView{}
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.requireActivePlan(tx, uid, planID)
		if err != nil {
			return err
		}
		days, err := s.plans.DaysByPlan(tx, planID)
		if err != nil {
			return err
		}
		var day *model.PlanDay
		for i := range days {
			if days[i].DayIndex == req.DayIndex {
				day = &days[i]
			}
		}
		if day == nil {
			return util.NewAppError(constants.CodeValidation, fmt.Sprintf("Plan[id=%d] add task failed: day_index %d out of range", planID, req.DayIndex), nil)
		}
		slot, err := uniqueCustomSlot()
		if err != nil {
			return err
		}
		cur, err := s.plans.CurrentVersion(tx, planID)
		if err != nil {
			return err
		}
		// 同日重复提交/双击去重：同一天已存在标题相同且未完成的自定义任务则直接返回，不再新建。
		planTasks, err := s.plans.TasksByPlan(tx, planID)
		if err != nil {
			return err
		}
		for _, ex := range planTasks {
			if ex.DayID == day.ID && ex.Source == constants.PlanTaskSourceCustom &&
				ex.Status == constants.PlanItemPending && ex.Title == req.Title {
				tt := ex
				*out = toTaskView(tt)
				s.logger.Info(constants.LogPlanTaskAdded, "plan_id", planID, "day_index", req.DayIndex, "dedup", true)
				return nil
			}
		}
		t := &model.PlanTask{DayID: day.ID, PlanID: planID, Slot: slot, Source: constants.PlanTaskSourceCustom, Title: req.Title, Content: req.Content, Status: constants.PlanItemPending, Version: cur.Version}
		if err = s.plans.CreateTask(tx, t); err != nil {
			return fmt.Errorf("PlanTask[day_id=%d] custom create failed: %w", day.ID, err)
		}
		// 新增待办可能让此前“已完成”的天重新打开。
		if err = s.refreshDayAndMaybeComplete(tx, p, day.ID); err != nil {
			return err
		}
		*out = toTaskView(*t)
		s.logger.Info(constants.LogPlanTaskAdded, "plan_id", planID, "day_index", req.DayIndex)
		return nil
	})
	return out, e
}

func (s *PlanService) SetDayNote(uid, planID uint, dayIndex int, note string) (*dto.PlanDayView, error) {
	out := &dto.PlanDayView{}
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.requireActivePlan(tx, uid, planID)
		if err != nil {
			return err
		}
		days, err := s.plans.DaysByPlan(tx, planID)
		if err != nil {
			return err
		}
		var day *model.PlanDay
		for i := range days {
			if days[i].DayIndex == dayIndex {
				day = &days[i]
			}
		}
		if day == nil {
			return util.NewAppError(constants.CodeValidation, fmt.Sprintf("Plan[id=%d] note failed: day_index %d out of range", planID, dayIndex), nil)
		}
		day.Note = note // 手写备注：重算永不覆盖
		if err = s.plans.UpdateDay(tx, day); err != nil {
			return fmt.Errorf("PlanDay[id=%d] note failed: %w", day.ID, err)
		}
		s.logger.Info(constants.LogPlanDayNoted, "plan_id", planID, "day_index", dayIndex)
		return s.fillDayView(tx, p, day, map[uint]int{}, out)
	})
	return out, e
}

// refreshDayAndMaybeComplete 重算单日完成状态；若七天全部完成则结束计划并冻结报告。
func (s *PlanService) refreshDayAndMaybeComplete(tx *gorm.DB, p *model.AdjustmentPlan, dayID uint) error {
	day, e := s.plans.DayByID(tx, dayID, p.ID)
	if e != nil {
		return e
	}
	tks, e := s.plans.TasksByDay(tx, dayID)
	if e != nil {
		return e
	}
	allDone, hasDone := len(tks) > 0, false
	for _, t := range tks {
		if t.Status == constants.PlanItemPending {
			allDone = false
			break
		}
		if t.Status == constants.PlanItemDone {
			hasDone = true
		}
	}
	newStatus := day.Status
	if allDone && hasDone {
		newStatus = constants.PlanDayCompleted
	} else {
		newStatus = constants.PlanItemPending
	}
	if newStatus != day.Status {
		day.Status = newStatus
		if e = s.plans.UpdateDay(tx, day); e != nil {
			return fmt.Errorf("PlanDay[id=%d] status failed: %w", day.ID, e)
		}
	}

	// 仅在用户动作导致七天全部完成时自动结束；取消/暂停不影响。
	if p.Status == constants.PlanStatusActive {
		days, e := s.plans.DaysByPlan(tx, p.ID)
		if e != nil {
			return e
		}
		complete := len(days) == constants.PlanLen && todayIndex(p.StartDate) >= constants.PlanLen-1
		for _, d := range days {
			if d.Status != constants.PlanDayCompleted {
				complete = false
			}
		}
		if complete {
			p.Status = constants.PlanStatusCompleted
			fin := time.Now()
			p.FinishedAt = &fin
			allTasks, e := s.plans.TasksByPlan(tx, p.ID)
			if e != nil {
				return e
			}
			p.ReportJSON = marshalReport(BuildReport(p, days, allTasks))
			if e = s.plans.UpdatePlan(tx, p); e != nil {
				return fmt.Errorf("Plan[id=%d] auto-complete failed: %w", p.ID, e)
			}
			s.logger.Info(constants.LogPlanAutoCompleted, "plan_id", p.ID)
		}
	}
	return nil
}

// ---- 读取 / 报告 ----

func (s *PlanService) GetActive(uid uint) (*dto.PlanView, error) {
	var view *dto.PlanView
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.plans.ActivePlan(tx, uid)
		if err != nil {
			return err
		}
		view, err = s.buildView(tx, p)
		return err
	})
	return view, e
}

func (s *PlanService) Get(uid, planID uint) (*dto.PlanView, error) {
	var view *dto.PlanView
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.plans.PlanByID(tx, planID, uid)
		if err != nil {
			return fmt.Errorf("Plan[id=%d] fetch failed: %w", planID, err)
		}
		view, err = s.buildView(tx, p)
		return err
	})
	return view, e
}

func (s *PlanService) List(uid uint) ([]dto.PlanSummaryView, error) {
	var out []dto.PlanSummaryView
	e := s.inLock(uid, func(tx *gorm.DB) error {
		ps, err := s.plans.ListPlans(tx, uid, 20)
		if err != nil {
			return err
		}
		out = []dto.PlanSummaryView{}
		for _, p := range ps {
			cur := 0
			if v, err := s.plans.CurrentVersion(tx, p.ID); err == nil {
				cur = v.Version
			}
			out = append(out, dto.PlanSummaryView{ID: p.ID, Status: p.Status, StartDate: p.StartDate.Format("2006-01-02"), EndDate: p.EndDate.Format("2006-01-02"), Current: cur, FinishedAt: p.FinishedAt, CreatedAt: p.CreatedAt})
		}
		return nil
	})
	return out, e
}

// Report：终态计划回读冻结的 ReportJSON；进行中实时计算。结论始终一致、可复现。
func (s *PlanService) Report(uid, planID uint) (*dto.PlanReport, error) {
	var rep *dto.PlanReport
	e := s.inLock(uid, func(tx *gorm.DB) error {
		p, err := s.plans.PlanByID(tx, planID, uid)
		if err != nil {
			return fmt.Errorf("Plan[id=%d] fetch failed: %w", planID, err)
		}
		if p.ReportJSON != "" && (p.Status == constants.PlanStatusCompleted || p.Status == constants.PlanStatusCancelled) {
			var frozen dto.PlanReport
			jerr := json.Unmarshal([]byte(p.ReportJSON), &frozen)
			if jerr == nil {
				rep = &frozen
				return nil
			}
			s.logger.Error(constants.LogPlanReportFallback, "plan_id", planID, "error", jerr)
		}
		days, tks, err := s.loadDaysTasks(tx, p.ID)
		if err != nil {
			return err
		}
		r := BuildReport(p, days, tks)
		rep = &r
		return nil
	})
	return rep, e
}

// VersionSnapshot 回看指定版本（含已被替换版本）当时冻结的建议与来源。
func (s *PlanService) VersionSnapshot(uid, planID, versionID uint) (*dto.PlanVersionSnapshotView, error) {
	var out *dto.PlanVersionSnapshotView
	e := s.inLock(uid, func(tx *gorm.DB) error {
		if _, err := s.plans.PlanByID(tx, planID, uid); err != nil {
			return fmt.Errorf("Plan[id=%d] fetch failed: %w", planID, err)
		}
		v, err := s.plans.VersionByID(tx, versionID, planID)
		if err != nil {
			return fmt.Errorf("PlanVersion[id=%d] fetch failed: %w", versionID, err)
		}
		var snap snapshotFile
		if err = json.Unmarshal([]byte(v.SnapshotJSON), &snap); err != nil {
			return fmt.Errorf("PlanVersion[id=%d] snapshot decode failed: %w", versionID, err)
		}
		days := make([]dto.PlanVersionDay, 0, len(snap.Days))
		for _, d := range snap.Days {
			tasks := make([]dto.PlanVersionTask, 0, len(d.Tasks))
			for _, t := range d.Tasks {
				tasks = append(tasks, dto.PlanVersionTask{Slot: t.Slot, Title: t.Title, Content: t.Content})
			}
			days = append(days, dto.PlanVersionDay{DayIndex: d.DayIndex, Theme: d.Theme, Tasks: tasks})
		}
		out = &dto.PlanVersionSnapshotView{ID: v.ID, Version: v.Version, Status: v.Status, Trigger: v.Trigger, InputHash: v.InputHash, CreatedAt: v.CreatedAt, ReplacedAt: v.ReplacedAt, Source: snap.Source, Days: days}
		return nil
	})
	return out, e
}

func (s *PlanService) loadDaysTasks(db *gorm.DB, planID uint) ([]model.PlanDay, []model.PlanTask, error) {
	days, e := s.plans.DaysByPlan(db, planID)
	if e != nil {
		return nil, nil, fmt.Errorf("PlanDay[plan_id=%d] list failed: %w", planID, e)
	}
	tks, e := s.plans.TasksByPlan(db, planID)
	if e != nil {
		return nil, nil, fmt.Errorf("PlanTask[plan_id=%d] list failed: %w", planID, e)
	}
	return days, tks, nil
}

func (s *PlanService) requireActivePlan(tx *gorm.DB, uid, planID uint) (*model.AdjustmentPlan, error) {
	p, e := s.plans.PlanByID(tx, planID, uid)
	if e != nil {
		return nil, fmt.Errorf("Plan[id=%d] fetch failed: %w", planID, e)
	}
	if p.Status != constants.PlanStatusActive {
		return nil, util.NewAppError(constants.CodeConflict, fmt.Sprintf("Plan[id=%d] operation failed: status=%s", planID, p.Status), nil)
	}
	return p, nil
}

// ---- 视图装配 ----

func (s *PlanService) buildView(db *gorm.DB, p *model.AdjustmentPlan) (*dto.PlanView, error) {
	days, tks, e := s.loadDaysTasks(db, p.ID)
	if e != nil {
		return nil, e
	}
	versions, e := s.plans.ListVersions(db, p.ID)
	if e != nil {
		return nil, e
	}
	verNo := map[uint]int{}
	versionViews := make([]dto.PlanVersionView, 0, len(versions))
	var source dto.PlanSourceView
	currentNo := 0
	for _, v := range versions {
		verNo[v.ID] = v.Version
		var snap snapshotFile
		_ = json.Unmarshal([]byte(v.SnapshotJSON), &snap)
		if v.Status == constants.PlanVersionCurrent {
			// 默认取当前版本冻结来源；随后若能读到实时数据则以实时统计覆盖，
			// 保证“来源统计与数据库实际条数一致”，即使该次重复提交没有产生新版本。
			source = snap.Source
			currentNo = v.Version
		}
		versionViews = append(versionViews, dto.PlanVersionView{ID: v.ID, Version: v.Version, Status: v.Status, Trigger: v.Trigger, InputHash: v.InputHash, CreatedAt: v.CreatedAt, ReplacedAt: v.ReplacedAt, DayCount: len(snap.Days)})
	}
	// 当前计划的来源按近 14 天实时数据计算：同日重复提交不产生新版本，
	// 但来源中的原始记录数仍应与数据库实际条数一致。历史/被替换版本回看仍读各自冻结快照。
	if live, lerr := s.loadSignal(db, p.UserID); lerr == nil {
		source = toSource(live)
	}

	byDay := map[uint][]model.PlanTask{}
	for _, t := range tks {
		byDay[t.DayID] = append(byDay[t.DayID], t)
	}
	dayViews := make([]dto.PlanDayView, 0, len(days))
	for _, d := range days {
		tasks := byDay[d.ID]
		tv := make([]dto.PlanTaskView, 0, len(tasks))
		for _, t := range tasks {
			tv = append(tv, toTaskView(t))
		}
		dayViews = append(dayViews, dto.PlanDayView{ID: d.ID, DayIndex: d.DayIndex, Date: d.DayDate.Format("2006-01-02"), Theme: d.Theme, Status: d.Status, Note: d.Note, Tasks: tv, Version: verNo[d.VersionID], CreatedAt: d.CreatedAt})
	}
	return &dto.PlanView{ID: p.ID, Status: p.Status, StartDate: p.StartDate.Format("2006-01-02"), EndDate: p.EndDate.Format("2006-01-02"), FinishedAt: p.FinishedAt, Current: currentNo, Source: source, Days: dayViews, Versions: versionViews, CreatedAt: p.CreatedAt}, nil
}

func (s *PlanService) fillDayView(tx *gorm.DB, p *model.AdjustmentPlan, day *model.PlanDay, verNo map[uint]int, out *dto.PlanDayView) error {
	tks, e := s.plans.TasksByDay(tx, day.ID)
	if e != nil {
		return e
	}
	tv := make([]dto.PlanTaskView, 0, len(tks))
	for _, t := range tks {
		tv = append(tv, toTaskView(t))
	}
	*out = dto.PlanDayView{ID: day.ID, DayIndex: day.DayIndex, Date: day.DayDate.Format("2006-01-02"), Theme: day.Theme, Status: day.Status, Note: day.Note, Tasks: tv, Version: verNo[day.VersionID], CreatedAt: day.CreatedAt}
	return nil
}

func toTaskView(t model.PlanTask) dto.PlanTaskView {
	return dto.PlanTaskView{ID: t.ID, DayID: t.DayID, Slot: t.Slot, Source: t.Source, Title: t.Title, Content: t.Content, Status: t.Status, UserContent: t.UserContent, Version: t.Version, CreatedAt: t.CreatedAt}
}

func toSource(sig PlanSignal) dto.PlanSourceView {
	summary := sig.Summary
	if summary == nil {
		summary = []string{}
	}
	return dto.PlanSourceView{WindowDays: sig.WindowDays, MoodCount: sig.MoodCount, RawMoodCount: sig.RawMoodCount, JournalCount: sig.JournalCount, AssessmentCount: sig.AssessmentCount, AvgMood: sig.AvgMood, DominantTag: sig.DominantTag, LatestResult: sig.LatestResult, Summary: summary, InputHash: sig.Fingerprint}
}

func todayIndex(start time.Time) int {
	now := truncateDay(time.Now())
	return int(now.Sub(truncateDay(start)).Hours() / 24)
}

func uniqueCustomSlot() (string, error) {
	b := make([]byte, 8)
	if _, e := rand.Read(b); e != nil {
		return "", fmt.Errorf("PlanTask[slot] generate failed: %w", e)
	}
	return constants.PlanTaskSourceCustom + "_" + hex.EncodeToString(b), nil
}
