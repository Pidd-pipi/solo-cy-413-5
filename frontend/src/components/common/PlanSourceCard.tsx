import {Descriptions,Statistic,Tag,Typography} from 'antd';
import {MOOD_EMOJI,MOOD_LABELS} from '../../constants/mood';
import {PLAN_TRIGGER_LABELS} from '../../constants/plan';
import type {PlanSource,PlanTrigger} from '../../types';

// PlanSourceCard 回看计划来源：近 14 天情绪/日记/测评构成与建议理由、输入指纹。
export function PlanSourceCard({source,trigger}:{source:PlanSource;trigger?:PlanTrigger}){
  return <div className="plan-source">
    <Typography.Paragraph type="secondary" style={{marginBottom:8}}>
      建议依据近 {source.window_days} 天记录生成{trigger?` · 触发：${PLAN_TRIGGER_LABELS[trigger]}`:''}
    </Typography.Paragraph>
    <Descriptions size="small" column={2}>
      <Descriptions.Item label="情绪记录"><Statistic value={source.mood_count} suffix="次"/></Descriptions.Item>
      <Descriptions.Item label="平均心情"><Statistic value={source.avg_mood?source.avg_mood.toFixed(1):'—'} suffix="/10"/></Descriptions.Item>
      <Descriptions.Item label="日记"><Statistic value={source.journal_count} suffix="篇"/></Descriptions.Item>
      <Descriptions.Item label="测评"><Statistic value={source.assessment_count} suffix="次"/></Descriptions.Item>
    </Descriptions>
    {source.dominant_tag && <Typography.Paragraph style={{marginTop:8}}>
      主导情绪：<Tag color="green">{MOOD_EMOJI[source.dominant_tag as keyof typeof MOOD_EMOJI]} {MOOD_LABELS[source.dominant_tag as keyof typeof MOOD_LABELS]}</Tag>
      {source.latest_result && <Tag>{source.latest_result}</Tag>}
    </Typography.Paragraph>}
    {source.summary.map((s,i)=><Typography.Paragraph key={i} style={{marginBottom:4}}>{s}</Typography.Paragraph>)}
    <Typography.Text type="secondary" style={{fontSize:12}}>版本输入指纹：<code>{source.input_hash.slice(0,16)}…</code>（输入不变则跳过重算）</Typography.Text>
  </div>;
}
