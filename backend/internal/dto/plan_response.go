package dto

import "time"

// ---- 计划详情响应 ----

type PlanTaskView struct {
	ID          uint      `json:"id"`
	DayID       uint      `json:"day_id"`
	Slot        string    `json:"slot"`
	Source      string    `json:"source"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	UserContent string    `json:"user_content"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

type PlanDayView struct {
	ID        uint           `json:"id"`
	DayIndex  int            `json:"day_index"`
	Date      string         `json:"date"`
	Theme     string         `json:"theme"`
	Status    string         `json:"status"`
	Note      string         `json:"note"`
	Tasks     []PlanTaskView `json:"tasks"`
	Version   int            `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
}

type PlanSourceView struct {
	WindowDays      int      `json:"window_days"`
	MoodCount       int      `json:"mood_count"`
	JournalCount    int      `json:"journal_count"`
	AssessmentCount int      `json:"assessment_count"`
	AvgMood         float64  `json:"avg_mood"`
	DominantTag     string   `json:"dominant_tag"`
	LatestResult    string   `json:"latest_result"`
	Summary         []string `json:"summary"`
	InputHash       string   `json:"input_hash"`
}

type PlanVersionView struct {
	ID         uint       `json:"id"`
	Version    int        `json:"version"`
	Status     string     `json:"status"`
	Trigger    string     `json:"trigger"`
	InputHash  string     `json:"input_hash"`
	CreatedAt  time.Time  `json:"created_at"`
	ReplacedAt *time.Time `json:"replaced_at,omitempty"`
	DayCount   int        `json:"day_count"`
}

type PlanView struct {
	ID         uint              `json:"id"`
	Status     string            `json:"status"`
	StartDate  string            `json:"start_date"`
	EndDate    string            `json:"end_date"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
	Current    int               `json:"current_version"`
	Source     PlanSourceView    `json:"source"`
	Days       []PlanDayView     `json:"days"`
	Versions   []PlanVersionView `json:"versions"`
	CreatedAt  time.Time         `json:"created_at"`
}

type PlanSummaryView struct {
	ID         uint       `json:"id"`
	Status     string     `json:"status"`
	StartDate  string     `json:"start_date"`
	EndDate    string     `json:"end_date"`
	Current    int        `json:"current_version"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ---- 历史曲线与报告（终态冻结） ----

type PlanReportDay struct {
	DayIndex       int     `json:"day_index"`
	Date           string  `json:"date"`
	Status         string  `json:"status"`
	TotalTasks     int     `json:"total_tasks"`
	DoneTasks      int     `json:"done_tasks"`
	CompletionRate float64 `json:"completion_rate"`
}

type PlanReport struct {
	PlanID         uint            `json:"plan_id"`
	Status         string          `json:"status"`
	StartDate      string          `json:"start_date"`
	EndDate        string          `json:"end_date"`
	FinishedAt     *string         `json:"finished_at,omitempty"`
	TotalTasks     int             `json:"total_tasks"`
	DoneTasks      int             `json:"done_tasks"`
	CompletionRate float64         `json:"completion_rate"`
	Days           []PlanReportDay `json:"days"`
	Summary        []string        `json:"summary"`
}

// PlanVersionSnapshotView 回看某个（含已被替换的）版本当时的完整建议。
type PlanVersionSnapshotView struct {
	ID         uint             `json:"id"`
	Version    int              `json:"version"`
	Status     string           `json:"status"`
	Trigger    string           `json:"trigger"`
	InputHash  string           `json:"input_hash"`
	CreatedAt  time.Time        `json:"created_at"`
	ReplacedAt *time.Time       `json:"replaced_at,omitempty"`
	Source     PlanSourceView   `json:"source"`
	Days       []PlanVersionDay `json:"days"`
}

type PlanVersionDay struct {
	DayIndex int               `json:"day_index"`
	Theme    string            `json:"theme"`
	Tasks    []PlanVersionTask `json:"tasks"`
}

type PlanVersionTask struct {
	Slot    string `json:"slot"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
