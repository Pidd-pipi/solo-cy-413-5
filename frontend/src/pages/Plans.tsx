import {Button,Card,Col,Empty,Popconfirm,Progress,Row,Space,Tag,Typography,message} from 'antd';
import {useCallback,useEffect,useState} from 'react';
import dayjs from 'dayjs';
import {actPlan,addPlanTask,createPlan,getActivePlan,getPlan,getPlanReport,listPlans,regeneratePlan,setPlanDayNote,setPlanTask} from '../api/plan';
import {PlanCompletionChart} from '../components/common/PlanCompletionChart';
import {PlanDayCard} from '../components/common/PlanDayCard';
import {PlanSourceCard} from '../components/common/PlanSourceCard';
import {PlanVersionTimeline} from '../components/common/PlanVersionTimeline';
import {EmptyState} from '../components/common/EmptyState';
import {PLAN_STATUS_COLOR,PLAN_STATUS_LABELS} from '../constants/plan';
import type {Plan,PlanReport,PlanSummary,PlanTask} from '../types';

export function Plans(){
  const [list,setList]=useState<PlanSummary[]>([]);
  const [plan,setPlan]=useState<Plan>();
  const [report,setReport]=useState<PlanReport>();
  const [busy,setBusy]=useState(false);
  const [loading,setLoading]=useState(true);

  const refreshList=useCallback(()=>listPlans().then(setList).catch(()=>setList([])),[]);
  const openPlan=useCallback(async(id:number)=>{
    setLoading(true);
    try{
      const p=await getPlan(id);setPlan(p);
      setReport(await getPlanReport(id));
    }catch(e){message.error((e as Error).message);}finally{setLoading(false);}
  },[]);

  const load=useCallback(async()=>{
    setLoading(true);
    try{
      const active=await getActivePlan();
      setPlan(active);setReport(await getPlanReport(active.id));
    }catch{ /* 没有进行中计划 */ setPlan(undefined); }
    finally{setLoading(false);refreshList();}
  },[refreshList]);

  useEffect(()=>{load();},[load]);

  const run=async(fn:()=>Promise<unknown>,ok?:string)=>{
    if(busy)return;
    setBusy(true);
    try{await fn();if(ok)message.success(ok);await load();}
    catch(e){message.error((e as Error).message);}
    finally{setBusy(false);}
  };

  const onCreate=()=>run(()=>createPlan(),'七日身心调整计划已生成');
  const onAction=(action:'pause'|'resume'|'complete'|'cancel',ok:string)=>
    plan&&run(()=>actPlan(plan.id,action),ok);
  const onRegen=()=>plan&&run(async()=>{
    const before=plan.current_version;
    const p=await regeneratePlan(plan.id);
    message.success(p.current_version>before?'已根据最新记录生成新版本':'近 14 天记录无变化，保持当前版本');
  });

  const readonly=!!plan&&plan.status!=='active';
  const doneTasks=plan?plan.days.reduce((n,d)=>n+d.tasks.filter(t=>t.status==='done').length,0):0;
  const totalTasks=plan?plan.days.reduce((n,d)=>n+d.tasks.length,0):0;
  const pct=totalTasks?Math.round(doneTasks/totalTasks*100):0;

  return <>
    <Typography.Title>七日身心调整计划</Typography.Title>
    <Typography.Paragraph type="secondary">
      系统依据近 14 天的情绪、日记与测评，为你生成 7 天的轻量身心建议。新记录会重算<strong>今天及以后、尚未完成</strong>的建议；
      已完成的任务、你手写的任务与备注永远不会被覆盖。
    </Typography.Paragraph>

    {!plan && <Card>
      <EmptyState title="你当前没有进行中的计划"/>
      <div style={{textAlign:'center',marginTop:12}}>
        <Button type="primary" size="large" loading={busy||loading} onClick={onCreate}>根据我的近况生成七日计划</Button>
      </div>
    </Card>}

    {plan && <>
      <Card style={{marginBottom:16}}>
        <Row gutter={16} align="middle">
          <Col flex="auto">
            <Space wrap>
              <Typography.Title level={3} style={{margin:0}}>第 {plan.current_version} 版计划</Typography.Title>
              <Tag color={PLAN_STATUS_COLOR[plan.status]}>{PLAN_STATUS_LABELS[plan.status]}</Tag>
            </Space>
            <Typography.Paragraph type="secondary" style={{margin:'4px 0 0'}}>
              {plan.start_date} ~ {plan.end_date} · 共 {totalTasks} 项任务，已完成 {doneTasks} 项
            </Typography.Paragraph>
            <Progress percent={pct} size="small" style={{maxWidth:360,marginTop:6}}/>
          </Col>
          <Col>
            <Space wrap>
              {plan.status==='active'&&<>
                <Button loading={busy} onClick={()=>onRegen()}>用最新记录重算</Button>
                <Button loading={busy} onClick={()=>onAction('pause','计划已暂停')}>暂停</Button>
                <Popconfirm title="提前结束计划？" onConfirm={()=>onAction('complete','计划已结束')}><Button loading={busy}>结束</Button></Popconfirm>
                <Popconfirm title="取消计划？已完成记录仍会保留。" okText="取消计划" onConfirm={()=>onAction('cancel','计划已取消')}><Button danger loading={busy}>取消</Button></Popconfirm>
              </>}
              {plan.status==='paused'&&<Button type="primary" loading={busy} onClick={()=>onAction('resume','计划已恢复')}>恢复</Button>}
              {(plan.status==='completed'||plan.status==='cancelled')&&<Space><Tag>已冻结历史报告</Tag><Button onClick={()=>load()}>回到我的计划</Button></Space>}
            </Space>
          </Col>
        </Row>
      </Card>

      <Row gutter={[16,16]}>
        <Col xs={24} lg={9}>
          <Card title="计划来源" size="small" style={{marginBottom:16}}>
            <PlanSourceCard source={plan.source}/>
          </Card>
          <Card title="历史完成曲线" size="small">
            {report?<PlanCompletionChart report={report}/>:<Empty/>}
            {report&&<Typography.Paragraph type="secondary" style={{marginTop:8,marginBottom:0}}>
              整体完成率 {Math.round(report.completion_rate*100)}%（{report.done_tasks}/{report.total_tasks}）
            </Typography.Paragraph>}
          </Card>
          <Card title="版本记录（被替换版本可回看）" size="small" style={{marginTop:16}}>
            <PlanVersionTimeline plan={plan}/>
          </Card>
        </Col>
        <Col xs={24} lg={15}>
          <Space direction="vertical" size={12} style={{width:'100%'}}>
            {plan.days.map(d=><PlanDayCard key={d.id} day={d} readonly={readonly} busy={busy}
              onToggleTask={(t:PlanTask)=>run(()=>setPlanTask(plan.id,t.id,{status:t.status==='done'?'pending':'done'}),'已保存')}
              onSaveContent={(t,user_content)=>run(()=>setPlanTask(plan.id,t.id,{user_content}),'感受已保存')}
              onSaveNote={note=>run(()=>setPlanDayNote(plan.id,d.day_index,note),'备注已保存')}
              onAddTask={(title,content)=>run(()=>addPlanTask(plan.id,{day_index:d.day_index,title,content}),'手写任务已添加')}/>)}
          </Space>
        </Col>
      </Row>
    </>}

    <Card title="历史计划" size="small" style={{marginTop:20}}>
      {list.filter(p=>!plan||p.id!==plan.id).length===0
        ? <EmptyState title="还没有历史计划"/>
        : <Space direction="vertical" style={{width:'100%'}}>
            {list.filter(p=>!plan||p.id!==plan.id).map(p=><Card key={p.id} size="small" hoverable
              onClick={()=>openPlan(p.id)}>
              <Space wrap>
                <Tag color={PLAN_STATUS_COLOR[p.status]}>{PLAN_STATUS_LABELS[p.status]}</Tag>
                <b>v{p.current_version}</b>
                <Typography.Text type="secondary">{p.start_date} ~ {p.end_date}</Typography.Text>
                {p.finished_at&&<Typography.Text type="secondary">结束于 {dayjs(p.finished_at).format('YYYY-MM-DD HH:mm')}</Typography.Text>}
              </Space>
            </Card>)}
          </Space>}
    </Card>
  </>;
}
