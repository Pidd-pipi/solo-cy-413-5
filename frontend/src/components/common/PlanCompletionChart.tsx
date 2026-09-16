import {Empty} from 'antd';
import {Bar,BarChart,CartesianGrid,ResponsiveContainer,Tooltip,XAxis,YAxis} from 'recharts';
import type {PlanReport} from '../../types';

interface RatePoint{name:string;rate:number;done:number;total:number}

// PlanCompletionChart 历史完成曲线：按天展示完成率（0-100%），终态与进行中回读同一组数据。
export function PlanCompletionChart({report,height=240}:{report:PlanReport;height?:number}){
  const data:RatePoint[]=report.days.map(d=>({name:`D${d.day_index+1}`,rate:Math.round(d.completion_rate*100),done:d.done_tasks,total:d.total_tasks}));
  if(!data.length) return <Empty description="还没有可绘制的完成数据"/>;
  const tip=(p:RatePoint|undefined)=>p?`${p.rate}%（${p.done}/${p.total}）`:'';
  return <div style={{width:'100%',height}}>
    <ResponsiveContainer>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--border,#eee)"/>
        <XAxis dataKey="name"/>
        <YAxis domain={[0,100]} unit="%"/>
        <Tooltip content={({active,payload})=>{
          if(!active||!payload||!payload.length) return null;
          const p=payload[0].payload as RatePoint;
          return <div style={{background:'#fff',border:'1px solid #eee',borderRadius:8,padding:'4px 8px',fontSize:12}}>
            {p.name} 完成率 {tip(p)}
          </div>;
        }}/>
        <Bar dataKey="rate" fill="var(--primary,#7ba47e)" radius={[6,6,0,0]}/>
      </BarChart>
    </ResponsiveContainer>
  </div>;
}
