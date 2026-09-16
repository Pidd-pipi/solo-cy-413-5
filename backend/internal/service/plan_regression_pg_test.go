package service_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 这是一组基于【真实 PostgreSQL 持久化】的可重复回归测试。
//
// 运行方式（已在 CI/本地准备好 Postgres 时）：
//
//	export TEST_DATABASE_URL="host=127.0.0.1 port=55432 user=postgres dbname=mindgarden_test sslmode=disable TimeZone=Asia/Shanghai"
//	go test ./internal/service/ -run TestPlanRegression -v
//
// 未设置 TEST_DATABASE_URL 时自动 skip，因此不影响无数据库环境下的普通 `go test ./...`。
// 可重复性：每个用例使用全新随机用户并在结尾删除其全部数据（用户/计划/版本/天/任务/情绪），
// 唯一约束（uk_plan_one_active 等）因此可以反复运行而不冲突。

var regressionLoc = func() *time.Location {
	if l, e := time.LoadLocation("Asia/Shanghai"); e == nil {
		return l
	}
	return time.UTC
}()

type planHarness struct {
	db  *gorm.DB
	svc *service.PlanService
}

func newPlanHarness(t *testing.T) *planHarness {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL 未设置，跳过基于真实 PostgreSQL 的计划回归测试")
	}
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if e != nil {
		t.Fatalf("[数据链路] 连接 PostgreSQL 失败: %v", e)
	}
	if e = db.AutoMigrate(
		&model.User{}, &model.Mood{}, &model.Journal{}, &model.Assessment{}, &model.UserAssessment{},
		&model.AdjustmentPlan{}, &model.PlanVersion{}, &model.PlanDay{}, &model.PlanTask{},
	); e != nil {
		t.Fatalf("[数据链路] AutoMigrate 失败: %v", e)
	}
	pr := repository.NewPlanRepository(db)
	if e = pr.EnsureIndexes(db); e != nil {
		t.Fatalf("[数据链路] 创建部分唯一索引失败: %v", e)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := service.NewPlanService(db, pr, repository.NewPlanDataRepository(db), logger)
	return &planHarness{db: db, svc: svc}
}

// createRegressionUser 创建随机用户并注册清理。返回其 uid。
func (h *planHarness) createUser(t *testing.T) uint {
	t.Helper()
	stamp := time.Now().UnixNano()
	u := &model.User{
		Email:        fmt.Sprintf("reg_%d@example.test", stamp),
		PasswordHash: "x",
		Nickname:     fmt.Sprintf("回归用户%d", stamp%100000),
		Role:         constants.RoleUser,
	}
	if e := h.db.Create(u).Error; e != nil {
		t.Fatalf("[数据链路] 创建测试用户失败: %v", e)
	}
	t.Cleanup(func() {
		// 按外键依赖顺序清理，保证重复运行不留数据。
		h.db.Exec("DELETE FROM plan_tasks WHERE plan_id IN (SELECT id FROM adjustment_plans WHERE user_id = ?)", u.ID)
		h.db.Exec("DELETE FROM plan_days WHERE plan_id IN (SELECT id FROM adjustment_plans WHERE user_id = ?)", u.ID)
		h.db.Exec("DELETE FROM plan_versions WHERE plan_id IN (SELECT id FROM adjustment_plans WHERE user_id = ?)", u.ID)
		h.db.Exec("DELETE FROM adjustment_plans WHERE user_id = ?", u.ID)
		h.db.Exec("DELETE FROM moods WHERE user_id = ?", u.ID)
		h.db.Exec("DELETE FROM journals WHERE user_id = ?", u.ID)
		h.db.Exec("DELETE FROM user_assessments WHERE user_id = ?", u.ID)
		h.db.Exec("DELETE FROM users WHERE id = ?", u.ID)
	})
	return u.ID
}

func mustMoodTags(t *testing.T, tags ...string) string {
	t.Helper()
	b, e := json.Marshal(tags)
	if e != nil {
		t.Fatalf("[测试夹具] 序列化情绪标签失败: %v", e)
	}
	return string(b)
}

// noonDate 返回相对今天 offset 天的“正午”，避免跨时区导致日期错位。
func noonDate(offset int) time.Time {
	y, m, d := time.Now().In(regressionLoc).Date()
	return time.Date(y, m, d, 12, 0, 0, 0, regressionLoc).AddDate(0, 0, offset)
}

func (h *planHarness) insertMood(t *testing.T, uid uint, level int, tags []string, at time.Time) {
	t.Helper()
	m := &model.Mood{UserID: uid, MoodLevel: level, MoodTags: mustMoodTags(t, tags...), Note: "", RecordDate: at}
	if e := h.db.Create(m).Error; e != nil {
		t.Fatalf("[数据链路] 写入情绪记录失败: %v", e)
	}
}

func (h *planHarness) dbCount(t *testing.T, table string, uid uint) int64 {
	t.Helper()
	var n int64
	if e := h.db.Table(table).Where("user_id = ?", uid).Count(&n).Error; e != nil {
		t.Fatalf("[数据链路] 统计 %s 失败: %v", table, e)
	}
	return n
}

func (h *planHarness) versionCount(t *testing.T, planID uint) int64 {
	t.Helper()
	var n int64
	if e := h.db.Model(&model.PlanVersion{}).Where("plan_id = ?", planID).Count(&n).Error; e != nil {
		t.Fatalf("[版本] 统计版本数失败: %v", e)
	}
	return n
}

// fireMoodChange 模拟“新情绪写入成功后”的自动重算钩子（与 MoodService.Create 中调用一致）。
func (h *planHarness) fireMoodChange(uid uint) {
	h.svc.OnSourceDataChanged(uid, constants.PlanTriggerMood)
}

// TestPlanRegressionDuplicateSourceRecords 覆盖需求的核心统计/版本/数据链路一致性。
func TestPlanRegressionDuplicateSourceRecords(t *testing.T) {
	h := newPlanHarness(t)
	uid := h.createUser(t)

	// 初始一条“今天、等级 6、calm”的情绪。
	h.insertMood(t, uid, 6, []string{constants.MoodCalm}, noonDate(0))

	view, isCreated, e := h.svc.Create(uid, "")
	if e != nil || !isCreated {
		t.Fatalf("[数据链路] 创建计划失败: created=%v err=%v", isCreated, e)
	}
	planID := view.ID
	baselineVersions := h.versionCount(t, planID)
	if baselineVersions != 1 {
		t.Fatalf("[版本] 新建计划应有 1 个版本，实际 %d", baselineVersions)
	}

	// 步骤 1：同日、同等级、同标签的重复提交（再写 2 条，共 3 条完全相同）。
	h.insertMood(t, uid, 6, []string{constants.MoodCalm}, noonDate(0))
	h.insertMood(t, uid, 6, []string{constants.MoodCalm}, noonDate(0))
	h.fireMoodChange(uid)

	active, e := h.svc.GetActive(uid)
	if e != nil {
		t.Fatalf("[数据链路] 重复提交后回读计划失败: %v", e)
	}
	if got := h.versionCount(t, planID); got != baselineVersions {
		t.Fatalf("[版本] 同日完全相同的重复提交不应产生新版本：期望版本数=%d，实际=%d", baselineVersions, got)
	}
	if active.Current != 1 {
		t.Fatalf("[版本] 当前版本号应保持 1，实际 %d", active.Current)
	}
	if active.Source.MoodCount != 1 {
		t.Fatalf("[统计] 有效观察数应为 1（重复折叠），实际 %d", active.Source.MoodCount)
	}
	if active.Source.RawMoodCount != 3 {
		t.Fatalf("[统计] 来源原始记录数应为 3，实际 %d", active.Source.RawMoodCount)
	}
	if dbN := h.dbCount(t, "moods", uid); dbN != 3 {
		t.Fatalf("[数据链路] 数据库实际情绪条数应为 3（原始记录必须保留），实际 %d", dbN)
	}
	if active.Source.RawMoodCount != int(h.dbCount(t, "moods", uid)) {
		t.Fatalf("[统计] 来源原始记录数(%d) 与数据库实际条数不一致", active.Source.RawMoodCount)
	}
	if active.Source.AvgMood != 6.0 {
		t.Fatalf("[统计] 平均心情不应被重复提交改变，期望 6.0，实际 %.2f", active.Source.AvgMood)
	}

	// 步骤 2：不同日期、内容完全相同（昨天、等级 6、calm）必须触发新版本。
	before := h.versionCount(t, planID)
	h.insertMood(t, uid, 6, []string{constants.MoodCalm}, noonDate(-1))
	h.fireMoodChange(uid)
	if got := h.versionCount(t, planID); got != before+1 {
		t.Fatalf("[版本] 不同日期的相同内容应触发新版本：期望 %d，实际 %d", before+1, got)
	}
	active2, _ := h.svc.GetActive(uid)
	if active2.Source.MoodCount != 2 || active2.Source.RawMoodCount != 4 {
		t.Fatalf("[统计] 跨日期后应为 2 条有效观察 / 4 条原始记录，实际 有效=%d 原始=%d",
			active2.Source.MoodCount, active2.Source.RawMoodCount)
	}
	if active2.Source.AvgMood != 6.0 {
		t.Fatalf("[统计] 两条等级 6 的有效观察平均应为 6.0，实际 %.2f", active2.Source.AvgMood)
	}

	// 步骤 3：同日但内容不同（今天、等级 3、anxious）必须触发新版本。
	before = h.versionCount(t, planID)
	h.insertMood(t, uid, 3, []string{constants.MoodAnxious}, noonDate(0))
	h.fireMoodChange(uid)
	if got := h.versionCount(t, planID); got != before+1 {
		t.Fatalf("[版本] 同日但等级/标签不同的记录应触发新版本：期望 %d，实际 %d", before+1, got)
	}
	active3, _ := h.svc.GetActive(uid)
	if active3.Source.MoodCount != 3 {
		t.Fatalf("[统计] 同日不同内容应计入有效观察，期望 3，实际 %d", active3.Source.MoodCount)
	}
	if active3.Source.RawMoodCount != int(h.dbCount(t, "moods", uid)) {
		t.Fatalf("[统计] 来源原始记录数(%d) 与数据库实际条数不一致", active3.Source.RawMoodCount)
	}
	if active3.Source.AvgMood != 5.0 { // (6+6+3)/3
		t.Fatalf("[统计] 三条有效观察平均应为 (6+6+3)/3=5.0，实际 %.2f", active3.Source.AvgMood)
	}

	// 步骤 4：再次写入与“今天 calm/6”完全相同的重复记录，验证在已有多种数据时仍幂等、不改均值。
	avgBefore := active3.Source.AvgMood
	before = h.versionCount(t, planID)
	h.insertMood(t, uid, 6, []string{constants.MoodCalm}, noonDate(0))
	h.fireMoodChange(uid)
	active4, _ := h.svc.GetActive(uid)
	if h.versionCount(t, planID) != before {
		t.Fatalf("[版本] 已有数据下的同日重复提交仍不应产生新版本，期望版本数 %d，实际 %d", before, h.versionCount(t, planID))
	}
	if active4.Source.AvgMood != avgBefore {
		t.Fatalf("[统计] 重复提交不应改变平均心情：期望 %.2f，实际 %.2f", avgBefore, active4.Source.AvgMood)
	}
	if active4.Source.MoodCount != 3 {
		t.Fatalf("[统计] 有效观察数应保持 3，实际 %d", active4.Source.MoodCount)
	}
	if active4.Source.RawMoodCount != int(h.dbCount(t, "moods", uid)) {
		t.Fatalf("[统计] 来源原始记录数(%d) 与数据库实际条数不一致", active4.Source.RawMoodCount)
	}
}

// TestPlanRegressionPreservesCompletedAndHandwritten 验证重算不破坏用户成果与手写内容。
func TestPlanRegressionPreservesCompletedAndHandwritten(t *testing.T) {
	h := newPlanHarness(t)
	uid := h.createUser(t)

	h.insertMood(t, uid, 7, []string{constants.MoodHappy}, noonDate(-2))
	view, _, e := h.svc.Create(uid, "")
	if e != nil {
		t.Fatalf("[数据链路] 创建计划失败: %v", e)
	}
	planID := view.ID
	day0 := view.Days[0]
	if len(day0.Tasks) == 0 {
		t.Fatal("[测试夹具] 第 1 天应至少有一个系统任务")
	}
	breathe := day0.Tasks[0]

	// 完成一个系统任务（已完成任务，重算不得删除/改写）。
	if _, e = h.svc.SetTask(uid, planID, breathe.ID, dto.PlanTaskRequest{Status: constants.PlanItemDone}); e != nil {
		t.Fatalf("[数据链路] 勾选系统任务失败: %v", e)
	}
	// 给该任务补充手写感受。
	if _, e = h.svc.SetTask(uid, planID, breathe.ID, dto.PlanTaskRequest{UserContent: "呼吸后确实平静了一些"}); e != nil {
		t.Fatalf("[数据链路] 保存任务手写感受失败: %v", e)
	}
	// 追加一个手写任务。
	custom, e := h.svc.AddTask(uid, planID, dto.PlanAddTaskRequest{DayIndex: 0, Title: "我自己的晨间拉伸", Content: "五分钟"})
	if e != nil {
		t.Fatalf("[数据链路] 追加手写任务失败: %v", e)
	}
	// 写一天备注。
	if _, e = h.svc.SetDayNote(uid, planID, 0, "今天想温柔一点"); e != nil {
		t.Fatalf("[数据链路] 保存天备注失败: %v", e)
	}

	// 触发一次“内容确实变化”的重算（昨天再写一条不同等级/标签的情绪）。
	before := h.versionCount(t, planID)
	h.insertMood(t, uid, 4, []string{constants.MoodTired, constants.MoodAnxious}, noonDate(-1))
	h.fireMoodChange(uid)
	if got := h.versionCount(t, planID); got != before+1 {
		t.Fatalf("[版本] 不同内容应触发一次重算新版本：期望 %d，实际 %d", before+1, got)
	}

	after, e := h.svc.GetActive(uid)
	if e != nil {
		t.Fatalf("[数据链路] 重算后回读计划失败: %v", e)
	}
	var d0 dto.PlanDayView
	for _, d := range after.Days {
		if d.DayIndex == 0 {
			d0 = d
		}
	}

	var gotBreathe *dto.PlanTaskView
	var gotCustom *dto.PlanTaskView
	for i := range d0.Tasks {
		tk := d0.Tasks[i]
		if tk.ID == breathe.ID {
			gotBreathe = &tk
		}
		if tk.ID == custom.ID {
			gotCustom = &tk
		}
	}
	if gotBreathe == nil {
		t.Fatal("[保留] 已完成的系统任务在重算后丢失（被错误删除）")
	}
	if gotBreathe.Status != constants.PlanItemDone {
		t.Fatalf("[保留] 已完成任务状态被覆盖：期望 done，实际 %s", gotBreathe.Status)
	}
	if gotBreathe.Title != breathe.Title {
		t.Fatalf("[保留] 已完成任务标题/建议被覆盖：期望 %q，实际 %q", breathe.Title, gotBreathe.Title)
	}
	if gotBreathe.UserContent != "呼吸后确实平静了一些" {
		t.Fatalf("[保留] 任务的手写感受被覆盖：实际 %q", gotBreathe.UserContent)
	}
	if gotCustom == nil {
		t.Fatal("[保留] 手写任务在重算后丢失（被错误删除）")
	}
	if gotCustom.Title != "我自己的晨间拉伸" || gotCustom.Source != constants.PlanTaskSourceCustom {
		t.Fatalf("[保留] 手写任务内容/来源被改动：%+v", gotCustom)
	}
	if d0.Note != "今天想温柔一点" {
		t.Fatalf("[保留] 天备注被覆盖：实际 %q", d0.Note)
	}

	// 历史冻结结论一致性：结束计划后，报告回读的完成统计要把保留下来的已完成任务计入。
	if _, e = h.svc.Action(uid, planID, "complete"); e != nil {
		t.Fatalf("[数据链路] 结束计划失败: %v", e)
	}
	rep, e := h.svc.Report(uid, planID)
	if e != nil {
		t.Fatalf("[数据链路] 读取历史报告失败: %v", e)
	}
	if rep.Status != constants.PlanStatusCompleted {
		t.Fatalf("[历史结论] 报告状态应为 completed，实际 %s", rep.Status)
	}
	if rep.DoneTasks < 1 {
		t.Fatalf("[历史结论] 已完成任务应计入冻结报告，实际完成数 %d", rep.DoneTasks)
	}
	if rep.Days[0].DoneTasks < 1 {
		t.Fatalf("[历史结论] 第 1 天完成数应包含保留的已完成任务，实际 %d", rep.Days[0].DoneTasks)
	}
}
