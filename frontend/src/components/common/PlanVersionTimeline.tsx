import {Button,Modal,Tag,Timeline,Typography} from 'antd';
import {useState} from 'react';
import {PLAN_TRIGGER_LABELS} from '../../constants/plan';
import {getPlanVersion} from '../../api/plan';
import type {Plan,PlanVersionSnapshot} from '../../types';
import {PlanSourceCard} from './PlanSourceCard';

// PlanVersionTimeline 回看所有版本（含被替换版本），点击可查看该版本当时的完整七日建议。
export function PlanVersionTimeline({plan}:{plan:Plan}){
  const [open,setOpen]=useState<number>();
  const [snap,setSnap]=useState<PlanVersionSnapshot>();
  const [loading,setLoading]=useState(false);

  const show=async(versionId:number)=>{
    setOpen(versionId);setSnap(undefined);setLoading(true);
    try{setSnap(await getPlanVersion(plan.id,versionId));}finally{setLoading(false);}
  };

  return <>
    <Timeline items={plan.versions.map(v=>({
      color:v.status==='current'?'green':'gray',
      children:<div>
        <Typography.Text strong>v{v.version}</Typography.Text>{' '}
        <Tag color={v.status==='current'?'green':'default'}>{v.status==='current'?'当前版本':'已被替换'}</Tag>
        <Tag>{PLAN_TRIGGER_LABELS[v.trigger]}</Tag>
        <Typography.Text type="secondary" style={{fontSize:12}}>{new Date(v.created_at).toLocaleString()}</Typography.Text>
        <div><Button type="link" size="small" style={{padding:0}} onClick={()=>show(v.id)}>查看该版本建议</Button></div>
      </div>
    }))}/>
    <Modal open={open!==undefined} footer={null} width={720} onCancel={()=>setOpen(undefined)}
      title={snap?`v${snap.version} 版本内容`:'加载中…'} confirmLoading={loading}>
      {snap && <>
        <PlanSourceCard source={snap.source} trigger={snap.trigger}/>
        {snap.days.map(d=><div key={d.day_index} style={{marginTop:12}}>
          <Typography.Text strong>第 {d.day_index+1} 天 · {d.theme}</Typography.Text>
          <ul>{d.tasks.map((t,i)=><li key={i}><b>{t.title}</b>：{t.content}</li>)}</ul>
        </div>)}
      </>}
    </Modal>
  </>;
}
