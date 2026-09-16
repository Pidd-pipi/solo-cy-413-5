import {request} from '../utils/request';
import type {Plan,PlanSummary,PlanReport,PlanVersionSnapshot} from '../types';

export const createPlan=()=>request<Plan>('/plans',{method:'POST',body:JSON.stringify({})});
export const getActivePlan=()=>request<Plan>('/plans/active');
export const listPlans=()=>request<PlanSummary[]>('/plans');
export const getPlan=(id:number)=>request<Plan>(`/plans/${id}`);
export const regeneratePlan=(id:number)=>request<Plan>(`/plans/${id}/regenerate`,{method:'POST'});
export const actPlan=(id:number,action:'pause'|'resume'|'complete'|'cancel')=>request<Plan>(`/plans/${id}/actions`,{method:'POST',body:JSON.stringify({action})});
export const getPlanReport=(id:number)=>request<PlanReport>(`/plans/${id}/report`);
export const getPlanVersion=(id:number,versionId:number)=>request<PlanVersionSnapshot>(`/plans/${id}/versions/${versionId}`);
export const addPlanTask=(id:number,payload:{day_index:number;title:string;content?:string})=>request(`/plans/${id}/tasks`,{method:'POST',body:JSON.stringify(payload)});
export const setPlanTask=(id:number,taskId:number,payload:{status?:string;user_content?:string})=>request(`/plans/${id}/tasks/${taskId}`,{method:'PUT',body:JSON.stringify(payload)});
export const setPlanDayNote=(id:number,dayIndex:number,note:string)=>request(`/plans/${id}/days/${dayIndex}/note`,{method:'PUT',body:JSON.stringify({note})});
