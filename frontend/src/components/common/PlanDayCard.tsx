import {Button,Card,Checkbox,Input,Space,Tag,Tooltip,Typography} from 'antd';
import {useState} from 'react';
import dayjs from 'dayjs';
import {PLAN_ITEM_STATUS_LABELS,PLAN_TASK_SOURCE_LABELS} from '../../constants/plan';
import type {PlanDay,PlanTask} from '../../types';

interface Props{
  day:PlanDay;
  readonly:boolean;
  onToggleTask:(t:PlanTask)=>void;
  onSaveContent:(t:PlanTask,content:string)=>void;
  onSaveNote:(note:string)=>void;
  onAddTask:(title:string,content:string)=>void;
  busy?:boolean;
}

export function PlanDayCard({day,readonly,onToggleTask,onSaveContent,onSaveNote,onAddTask,busy}:Props){
  const [note,setNote]=useState(day.note);
  const [title,setTitle]=useState('');
  const [content,setContent]=useState('');
  const [adding,setAdding]=useState(false);
  const isToday=dayjs().format('YYYY-MM-DD')===day.date;
  const done=day.tasks.filter(t=>t.status==='done').length;

  return <Card size="small" className={`plan-day ${day.status==='completed'?'is-done':''} ${isToday?'is-today':''}`}
    title={<Space wrap><span>第 {day.day_index+1} 天</span><span style={{fontWeight:400,color:'#888'}}>{day.date}</span>
      {isToday && <Tag color="green">今天</Tag>}
      {day.status==='completed' && <Tag color="blue">已完成 {done}/{day.tasks.length}</Tag>}
    </Space>}
    extra={<Typography.Text type="secondary">{day.theme}</Typography.Text>}>
    {day.tasks.length===0 && <EmptyState/>}
    <Space direction="vertical" style={{width:'100%'}} size={10}>
      {day.tasks.map(t=>{
        const checked=t.status==='done';
        return <div key={t.id} className={`plan-task ${checked?'is-done':''}`}>
          <Space align="start" style={{width:'100%'}}>
            <Checkbox checked={checked} disabled={readonly||busy}
              onChange={()=>onToggleTask(t)}>
              <b>{t.title}</b>
            </Checkbox>
            <Tag color={t.source==='custom'?'purple':'default'}>{PLAN_TASK_SOURCE_LABELS[t.source]}</Tag>
          </Space>
          <Typography.Paragraph type="secondary" style={{margin:'2px 0 2px 24px',fontSize:13}}>{t.content}</Typography.Paragraph>
          <Input.TextArea key={t.id+'_'+t.status} defaultValue={t.user_content} placeholder="完成后写一句感受（手写内容重算不会被覆盖）"
            disabled={readonly} autoSize={{minRows:1,maxRows:4}} style={{marginLeft:24,maxWidth:'92%'}}
            onBlur={e=>{if(e.target.value.trim() && e.target.value!==t.user_content)onSaveContent(t,e.target.value.trim())}}/>
          {checked && <Typography.Text type="success" style={{marginLeft:24,fontSize:12}}>{PLAN_ITEM_STATUS_LABELS.done}</Typography.Text>}
        </div>;
      })}
    </Space>

    {!readonly && adding && <Card size="small" style={{marginTop:10,background:'#fafafa'}}>
      <Input placeholder="手写任务标题" value={title} onChange={e=>setTitle(e.target.value)} style={{marginBottom:8}}/>
      <Input.TextArea placeholder="具体打算怎么做（可选）" value={content} onChange={e=>setContent(e.target.value)} autoSize={{minRows:1,maxRows:3}} style={{marginBottom:8}}/>
      <Space><Button type="primary" size="small" disabled={!title.trim()||busy}
        onClick={()=>{onAddTask(title.trim(),content.trim());setTitle('');setContent('');setAdding(false);}}>保存手写任务</Button>
      <Button size="small" onClick={()=>setAdding(false)}>取消</Button></Space>
    </Card>}

    <div style={{marginTop:10}}>
      <Input.TextArea key={day.id+'_note'} defaultValue={day.note} placeholder="这一天的备注（手写，重算不覆盖）"
        disabled={readonly} autoSize={{minRows:1,maxRows:4}} value={note} onChange={e=>setNote(e.target.value)}
        onBlur={()=>{if(note!==day.note)onSaveNote(note)}}/>
    </div>
    {!readonly && !adding && <Tooltip title="为这一天追加一个自己的任务，重算也不会被删除">
      <Button type="dashed" size="small" icon={<span style={{fontWeight:700}}>＋</span>} style={{marginTop:8}} onClick={()=>setAdding(true)}>手写任务</Button>
    </Tooltip>}
  </Card>;
}

function EmptyState(){return <Typography.Text type="secondary">这一天暂无可执行任务。</Typography.Text>;}
