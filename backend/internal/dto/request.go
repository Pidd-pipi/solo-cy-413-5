package dto

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Nickname string `json:"nickname" validate:"required,min=2,max=40"`
}
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname" validate:"required,min=2,max=40"`
	Avatar    string `json:"avatar"`
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender" validate:"omitempty,oneof=male female other"`
}
type MoodRequest struct {
	MoodLevel  int      `json:"mood_level" validate:"required,min=1,max=10"`
	MoodTags   []string `json:"mood_tags" validate:"required,min=1,max=5"`
	Note       string   `json:"note" validate:"max=500"`
	RecordDate string   `json:"record_date" validate:"required,datetime=2006-01-02"`
}
type AssessmentRequest struct {
	Title       string `json:"title" validate:"required,max=100"`
	Description string `json:"description" validate:"max=500"`
	Category    string `json:"category" validate:"required"`
	Questions   string `json:"questions" validate:"required"`
	ScoringRule string `json:"scoring_rule" validate:"required"`
}
type TakeAssessmentRequest struct {
	Answers []int `json:"answers" validate:"required,min=1"`
}
type JournalRequest struct {
	Title     string `json:"title" validate:"required,max=120"`
	Content   string `json:"content" validate:"required,max=5000"`
	MoodLevel int    `json:"mood_level" validate:"min=1,max=10"`
	Weather   string `json:"weather" validate:"max=30"`
	IsPrivate bool   `json:"is_private"`
}

// PlanCreateRequest 由前端显式发起；幂等键可选，用于并发/双击去重。
type PlanCreateRequest struct {
	IdempotencyKey string `json:"idempotency_key" validate:"max=64"`
}

// PlanAddTaskRequest 给某天追加一条手写任务（自定义任务永不被重算覆盖）。
type PlanAddTaskRequest struct {
	DayIndex int    `json:"day_index" validate:"min=0,max=6"`
	Title    string `json:"title" validate:"required,max=120"`
	Content  string `json:"content" validate:"max=1000"`
}

// PlanTaskRequest 更新任务完成状态或补充手写感受。
type PlanTaskRequest struct {
	Status      string `json:"status" validate:"omitempty,oneof=pending done skipped"`
	UserContent string `json:"user_content" validate:"max=2000"`
}

// PlanDayNoteRequest 给某天写/改备注（手写内容，重算不覆盖）。
type PlanDayNoteRequest struct {
	Note string `json:"note" validate:"max=2000"`
}

// PlanActionRequest 暂停/恢复/结束/取消，action 为状态机动作。
type PlanActionRequest struct {
	Action string `json:"action" validate:"required,oneof=pause resume complete cancel"`
}
