package constants

// 计划生命周期状态。
const (
	PlanStatusActive    = "active"    // 进行中（含被暂停的由 PlanDay/单独状态表达）
	PlanStatusPaused    = "paused"    // 已暂停，可恢复
	PlanStatusCompleted = "completed" // 已结束（七天全部完成）
	PlanStatusCancelled = "cancelled" // 已取消
)

// PlanActiveStatuses 占用“一个进行中名额”的状态。
var PlanActiveStatuses = []string{PlanStatusActive, PlanStatusPaused}

// 版本状态。
const (
	PlanVersionCurrent  = "current"
	PlanVersionReplaced = "replaced"
)

// 版本触发来源。
const (
	PlanTriggerInit       = "init"       // 新建计划
	PlanTriggerManual     = "manual"     // 用户手动重算
	PlanTriggerMood       = "mood"       // 新情绪记录触发
	PlanTriggerJournal    = "journal"    // 新日记触发
	PlanTriggerAssessment = "assessment" // 新测评提交触发
)

// 天 / 任务状态。
const (
	PlanItemPending  = "pending"
	PlanItemDone     = "done"
	PlanItemSkipped  = "skipped"
	PlanDayCompleted = "completed"
)

// 任务来源。
const (
	PlanTaskSourceSystem = "system"
	PlanTaskSourceCustom = "custom"
)

// 系统任务槽位：重算时按槽位替换，便于去重与展示。
const (
	PlanSlotBreathe = "breathe" // 呼吸 / 放松
	PlanSlotMove    = "move"    // 身体活动
	PlanSlotWind    = "wind"    // 睡前放松
	PlanSlotConnect = "connect" // 人际连接 / 觉察
)

// PlanSlots 每日系统任务的槽位顺序（部分天用 connect 替换 wind）。
var PlanSlots = []string{PlanSlotBreathe, PlanSlotMove, PlanSlotWind}

// PlanLen 计划天数。
const PlanLen = 7
