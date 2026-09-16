package service

// PlanChangeHook 在情绪 / 日记 / 测评写入成功后触发计划重算。
// 由 PlanService 实现；为避免包循环，源服务只依赖该接口并通过 Setter 注入。
type PlanChangeHook interface {
	OnSourceDataChanged(userID uint, trigger string)
}

// notifyPlan 在原始记录已提交后触发；hook 内部自行处理错误，绝不影响原始写入。
func notifyPlan(hook PlanChangeHook, userID uint, trigger string) {
	if hook != nil {
		hook.OnSourceDataChanged(userID, trigger)
	}
}
