import type {PlanStatus,PlanTrigger,PlanItemStatus,PlanTaskSource} from '../types';

export const PLAN_STATUS_LABELS:Record<PlanStatus,string>={active:'进行中',paused:'已暂停',completed:'已结束',cancelled:'已取消'};
export const PLAN_STATUS_COLOR:Record<PlanStatus,string>={active:'green',paused:'gold',completed:'blue',cancelled:'default'};
export const PLAN_TRIGGER_LABELS:Record<PlanTrigger,string>={init:'初次生成',manual:'手动重算',mood:'新情绪记录',journal:'新日记',assessment:'新测评结果'};
export const PLAN_ITEM_STATUS_LABELS:Record<PlanItemStatus,string>={pending:'待完成',done:'已完成',skipped:'已跳过',completed:'本日完成'};
export const PLAN_TASK_SOURCE_LABELS:Record<PlanTaskSource,string>={system:'建议',custom:'手写'};
export const PLAN_LEN=7;
