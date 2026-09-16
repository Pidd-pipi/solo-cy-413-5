package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/dto"
	"github.com/blueship581/mindgarden/backend/internal/model"
)

func tagsJSON(t *testing.T, tags ...string) string {
	t.Helper()
	b, e := json.Marshal(tags)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func day(offset int) time.Time {
	return time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC).AddDate(0, 0, offset)
}

func TestGeneratePlanShape(t *testing.T) {
	sig := BuildSignal(nil, nil, nil)
	days := GeneratePlan(sig)
	if len(days) != constants.PlanLen {
		t.Fatalf("want %d days, got %d", constants.PlanLen, len(days))
	}
	for i, d := range days {
		if d.DayIndex != i {
			t.Errorf("day index mismatch: want %d got %d", i, d.DayIndex)
		}
		if d.Theme == "" || len(d.Tasks) == 0 {
			t.Errorf("day %d must have theme and tasks", i)
		}
		for _, tk := range d.Tasks {
			if tk.Slot == "" || tk.Title == "" {
				t.Errorf("day %d task missing slot/title", i)
			}
		}
	}
}

func TestFingerprintDeterministic(t *testing.T) {
	moods := []model.Mood{
		{ID: 1, MoodLevel: 3, MoodTags: tagsJSON(t, constants.MoodAnxious, constants.MoodTired), RecordDate: day(-1)},
		{ID: 2, MoodLevel: 4, MoodTags: tagsJSON(t, constants.MoodTired), RecordDate: day(-3)},
	}
	a := BuildSignal(moods, nil, nil)
	b := BuildSignal(moods, nil, nil)
	if a.Fingerprint != b.Fingerprint {
		t.Fatal("same input must yield same fingerprint (idempotent recompute)")
	}
	// 确定性：同一信号多次生成的七日计划逐字一致。
	pa, pb := GeneratePlan(a), GeneratePlan(b)
	for i := range pa {
		if pa[i].Theme != pb[i].Theme || len(pa[i].Tasks) != len(pb[i].Tasks) {
			t.Fatalf("day %d plan not deterministic", i)
		}
	}
}

func TestDuplicateSameDaySubmissionDoesNotChangeSignal(t *testing.T) {
	one := []model.Mood{
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
	}
	// 同账号、同一天、等级与标签完全相同的重复提交（共 3 条）。
	dup := []model.Mood{
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
	}
	a, b := BuildSignal(one, nil, nil), BuildSignal(dup, nil, nil)
	if a.Fingerprint != b.Fingerprint {
		t.Fatal("identical same-day duplicates must not change the fingerprint")
	}
	if b.MoodCount != 1 || b.AvgMood != 6 || b.RawMoodCount != 3 {
		t.Fatalf("want 1 effective obs (avg 6) but 3 raw records, got count=%d avg=%.1f raw=%d", b.MoodCount, b.AvgMood, b.RawMoodCount)
	}
	if b.TagCounts[constants.MoodCalm] != 1 {
		t.Fatalf("duplicate calm submissions must be counted once, got %d", b.TagCounts[constants.MoodCalm])
	}
	if a.MoodLow != b.MoodLow || a.Anxious != b.Anxious {
		t.Fatal("duplicate submissions must not change threshold signals")
	}
}

func TestSameContentOnDifferentDaysStillCounts(t *testing.T) {
	// 不同日期、内容完全相同：应作为两条独立有效观察参与重算。
	moods := []model.Mood{
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(-1)},
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
	}
	s := BuildSignal(moods, nil, nil)
	if s.MoodCount != 2 {
		t.Fatalf("different-day records must both count, got %d", s.MoodCount)
	}
	if BuildSignal(moods[:1], nil, nil).Fingerprint == s.Fingerprint {
		t.Fatal("adding a same-content record on a different day must change the fingerprint")
	}
}

func TestDifferentContentSameDayStillCounts(t *testing.T) {
	// 同一天、等级不同：各自保留。
	moods := []model.Mood{
		{MoodLevel: 6, MoodTags: tagsJSON(t, constants.MoodCalm), RecordDate: day(0)},
		{MoodLevel: 3, MoodTags: tagsJSON(t, constants.MoodAnxious), RecordDate: day(0)},
	}
	s := BuildSignal(moods, nil, nil)
	if s.MoodCount != 2 {
		t.Fatalf("same-day different-content records must both count, got %d", s.MoodCount)
	}
	if s.AvgMood != 4.5 {
		t.Fatalf("want avg 4.5, got %.2f", s.AvgMood)
	}
}

func TestTagOrderInsensitiveDedupe(t *testing.T) {
	a := BuildSignal([]model.Mood{
		{MoodLevel: 5, MoodTags: tagsJSON(t, constants.MoodAnxious, constants.MoodTired), RecordDate: day(0)},
	}, nil, nil)
	b := BuildSignal([]model.Mood{
		{MoodLevel: 5, MoodTags: tagsJSON(t, constants.MoodTired, constants.MoodAnxious), RecordDate: day(0)},
		{MoodLevel: 5, MoodTags: tagsJSON(t, constants.MoodTired, constants.MoodAnxious), RecordDate: day(0)},
	}, nil, nil)
	if a.Fingerprint != b.Fingerprint || b.MoodCount != 1 {
		t.Fatal("same tags in different order must be equivalent and deduped")
	}
}

func TestDifferentInputChangesFingerprint(t *testing.T) {
	base := BuildSignal([]model.Mood{{MoodLevel: 7, MoodTags: tagsJSON(t, constants.MoodHappy), RecordDate: day(-1)}}, nil, nil)
	lowMoods := []model.Mood{
		{MoodLevel: 2, MoodTags: tagsJSON(t, constants.MoodAnxious), RecordDate: day(-1)},
		{MoodLevel: 3, MoodTags: tagsJSON(t, constants.MoodAnxious), RecordDate: day(-2)},
	}
	low := BuildSignal(lowMoods, nil, nil)
	if base.Fingerprint == low.Fingerprint {
		t.Fatal("different mood data must change the fingerprint")
	}
	if !low.MoodLow || !low.Anxious {
		t.Fatal("low average + anxious tags should set MoodLow/Anxious signals")
	}
}

func TestBuildReportCountsOnlyDoneAndIsStable(t *testing.T) {
	start := day(-6)
	p := &model.AdjustmentPlan{ID: 1, UserID: 7, Status: constants.PlanStatusCompleted, StartDate: start, EndDate: start.AddDate(0, 0, 6)}
	days := []model.PlanDay{}
	tasks := []model.PlanTask{}
	for i := 0; i < constants.PlanLen; i++ {
		d := model.PlanDay{ID: uint(i + 1), PlanID: 1, DayIndex: i, DayDate: start.AddDate(0, 0, i), Status: constants.PlanItemPending}
		days = append(days, d)
		// 每天 2 个任务：第 0、1 天各完成 1 个，其余全 pending。
		st0, st1 := constants.PlanItemPending, constants.PlanItemPending
		if i <= 1 {
			st0 = constants.PlanItemDone
			d.Status = constants.PlanDayCompleted
		}
		tasks = append(tasks,
			model.PlanTask{ID: uint(i*2 + 1), DayID: d.ID, PlanID: 1, Status: st0},
			model.PlanTask{ID: uint(i*2 + 2), DayID: d.ID, PlanID: 1, Status: st1},
		)
	}
	rep := BuildReport(p, days, tasks)
	if rep.TotalTasks != 14 || rep.DoneTasks != 2 {
		t.Fatalf("want 14 total / 2 done, got %d / %d", rep.TotalTasks, rep.DoneTasks)
	}
	// 跨账号任务不应计入：报告只按传入的本账号天/任务聚合，这里重复调用结论必须一致。
	again := BuildReport(p, days, tasks)
	if again.CompletionRate != rep.CompletionRate || again.DoneTasks != rep.DoneTasks {
		t.Fatal("report must be reproducible (history reads same conclusion)")
	}
	var _ dto.PlanReport = rep
}
