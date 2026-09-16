export type MoodTag='happy'|'anxious'|'tired'|'angry'|'calm'; export type AssessmentCategory='anxiety'|'depression'|'stress'|'sleep';
export interface User {id:number;email:string;nickname:string;avatar:string;birth_date?:string;gender:string;role:string;created_at:string}
export interface Mood {id:number;user_id:number;mood_level:number;mood_tags:string;note:string;record_date:string;created_at:string}
export interface Assessment {id:number;title:string;description:string;category:AssessmentCategory;questions:string;scoring_rule:string}
export interface Journal {id:number;title:string;content:string;mood_level:number;weather:string;is_private:boolean;created_at:string;updated_at:string}
export interface UserAssessment {id:number;assessment_id:number;score:number;result:string;suggestion:string;created_at:string}
export interface ApiResponse<T>{code:number;message:string;data:T}

// ---- 七日身心调整计划 ----
export type PlanStatus='active'|'paused'|'completed'|'cancelled';
export type PlanItemStatus='pending'|'done'|'skipped'|'completed';
export type PlanTaskSource='system'|'custom';
export type PlanTrigger='init'|'manual'|'mood'|'journal'|'assessment';

export interface PlanTask{id:number;day_id:number;slot:string;source:PlanTaskSource;title:string;content:string;status:PlanItemStatus;user_content:string;version:number;created_at:string}
export interface PlanDay{id:number;day_index:number;date:string;theme:string;status:PlanItemStatus;note:string;tasks:PlanTask[];version:number;created_at:string}
export interface PlanSource{window_days:number;mood_count:number;journal_count:number;assessment_count:number;avg_mood:number;dominant_tag:string;latest_result:string;summary:string[];input_hash:string}
export interface PlanVersion{id:number;version:number;status:'current'|'replaced';trigger:PlanTrigger;input_hash:string;created_at:string;replaced_at?:string;day_count:number}
export interface Plan{id:number;status:PlanStatus;start_date:string;end_date:string;finished_at?:string;current_version:number;source:PlanSource;days:PlanDay[];versions:PlanVersion[];created_at:string}
export interface PlanSummary{id:number;status:PlanStatus;start_date:string;end_date:string;current_version:number;finished_at?:string;created_at:string}
export interface PlanReportDay{day_index:number;date:string;status:PlanItemStatus;total_tasks:number;done_tasks:number;completion_rate:number}
export interface PlanReport{plan_id:number;status:PlanStatus;start_date:string;end_date:string;finished_at?:string;total_tasks:number;done_tasks:number;completion_rate:number;days:PlanReportDay[];summary:string[]}
export interface PlanVersionTask{slot:string;title:string;content:string}
export interface PlanVersionDay{day_index:number;theme:string;tasks:PlanVersionTask[]}
export interface PlanVersionSnapshot{id:number;version:number;status:string;trigger:PlanTrigger;input_hash:string;created_at:string;replaced_at?:string;source:PlanSource;days:PlanVersionDay[]}
