package service

import (
	"encoding/json"
	"fmt"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"sort"
	"strings"
	"time"
)

// PlanSignal 是近 14 天数据提炼出的建议信号；计划内容只由它决定，保证可复现、可回读。
type PlanSignal struct {
	WindowDays      int
	MoodCount       int
	AvgMood         float64
	LatestMood      int
	TagCounts       map[string]int
	DominantTag     string
	JournalCount    int
	AssessmentCount int
	LatestResult    string
	MoodLow         bool
	Anxious         bool
	Tired           bool
	Angry           bool
	StressHigh      bool
	Summary         []string
	Fingerprint     string
}

// RecTask 引擎生成的单条系统任务建议。
type RecTask struct {
	Slot    string
	Title   string
	Content string
}

// RecDay 一天的建议。
type RecDay struct {
	DayIndex int
	Theme    string
	Tasks    []RecTask
}

const planWindowDays = 14

// BuildSignal 从近 14 天的情绪/日记/测评中提取信号并计算稳定指纹。
// 指纹对“同日完全相同的重复提交”去重，因此重复数据不会触发新版本或重复任务。
func BuildSignal(moods []model.Mood, journals []model.Journal, assessments []model.UserAssessment) PlanSignal {
	sig := PlanSignal{WindowDays: planWindowDays, TagCounts: map[string]int{}}

	sum, dedupMood := 0, map[string]bool{}
	var moodLines []string
	for _, m := range moods {
		sum += m.MoodLevel
		for _, t := range parseMoodTags(m.MoodTags) {
			if containsStr(constants.MoodTags, t) {
				sig.TagCounts[t]++
			}
		}
		line := fmt.Sprintf("%s:%d:%s", m.RecordDate.Format("2006-01-02"), m.MoodLevel, strings.Join(parseMoodTags(m.MoodTags), ","))
		if !dedupMood[line] {
			dedupMood[line] = true
			moodLines = append(moodLines, line)
		}
	}
	sig.MoodCount = len(moods)
	if len(moods) > 0 {
		sig.AvgMood = float64(sum) / float64(len(moods))
		sig.LatestMood = moods[len(moods)-1].MoodLevel
	}

	journalLines, dedupJournal := []string{}, map[string]bool{}
	for _, j := range journals {
		line := fmt.Sprintf("%s:%d:%s", j.CreatedAt.Format("2006-01-02"), j.MoodLevel, j.Title)
		if !dedupJournal[line] {
			dedupJournal[line] = true
			journalLines = append(journalLines, line)
		}
	}
	sig.JournalCount = len(journals)

	assessmentLines, dedupAssess := []string{}, map[string]bool{}
	for _, a := range assessments {
		if strings.Contains(a.Result, "关照") {
			sig.StressHigh = true
		}
		sig.LatestResult = a.Result
		line := fmt.Sprintf("%s:%d", a.CreatedAt.Format("2006-01-02"), a.Score)
		if !dedupAssess[line] {
			dedupAssess[line] = true
			assessmentLines = append(assessmentLines, line)
		}
	}
	sig.AssessmentCount = len(assessments)

	best := 0
	for _, t := range constants.MoodTags {
		if sig.TagCounts[t] > best {
			best = sig.TagCounts[t]
			sig.DominantTag = t
		}
	}
	sig.Anxious = sig.TagCounts[constants.MoodAnxious] >= 2
	sig.Tired = sig.TagCounts[constants.MoodTired] >= 2
	sig.Angry = sig.TagCounts[constants.MoodAngry] >= 2
	sig.MoodLow = sig.MoodCount > 0 && sig.AvgMood <= 4.5

	sig.Summary = buildSummary(sig)

	sort.Strings(moodLines)
	sort.Strings(journalLines)
	sort.Strings(assessmentLines)
	sig.Fingerprint = util.HashStable(
		"v1",
		fmt.Sprintf("avg=%.2f", sig.AvgMood),
		"moods="+strings.Join(moodLines, ";"),
		"journals="+strings.Join(journalLines, ";"),
		"assess="+strings.Join(assessmentLines, ";"),
	)
	return sig
}

func buildSummary(s PlanSignal) []string {
	out := []string{}
	if s.MoodCount == 0 {
		out = append(out, "近 14 天还没有情绪记录，计划从温和的作息与觉察开始，陪你建立节奏。")
	} else {
		out = append(out, fmt.Sprintf("近 14 天记录了 %d 次情绪，平均心情 %.1f/10。", s.MoodCount, s.AvgMood))
	}
	switch {
	case s.StressHigh:
		out = append(out, "最近一次测评提示需要更多关照，计划以减压与休息为主，并在困扰持续时建议寻求专业支持。")
	case s.MoodLow:
		out = append(out, "整体心情偏低，计划安排轻量活动与稳定作息，避免给自己设过高目标。")
	case s.Anxious:
		out = append(out, "焦虑出现较频繁，计划加入呼吸与着陆练习，帮助安定神经系统。")
	case s.Tired:
		out = append(out, "疲惫感较多，计划优先睡眠与恢复，运动量保持轻柔。")
	case s.Angry:
		out = append(out, "有一些愤怒情绪，计划用温和运动与表达练习释放张力。")
	}
	if s.JournalCount > 0 {
		out = append(out, fmt.Sprintf("你写了 %d 篇日记，计划延续这种自我表达，并安排一次回顾。", s.JournalCount))
	}
	return out
}

// GeneratePlan 依据信号生成确定性的七日建议（同一天同一信号结果恒定）。
func GeneratePlan(sig PlanSignal) []RecDay {
	days := make([]RecDay, 0, constants.PlanLen)
	for i := 0; i < constants.PlanLen; i++ {
		days = append(days, RecDay{
			DayIndex: i,
			Theme:    dayTheme(sig, i),
			Tasks:    dayTasks(sig, i),
		})
	}
	return days
}

func dayTheme(sig PlanSignal, i int) string {
	base := []string{"安顿与觉察", "呼吸与放松", "轻量身体活动", "睡眠恢复", "情绪表达", "人际连接", "回顾与巩固"}
	switch {
	case sig.StressHigh && i == 6:
		return "温柔收尾与求助资源"
	case sig.MoodLow && (i == 1 || i == 3):
		return "低耗能自我关照"
	case sig.Tired && i == 2:
		return "恢复性拉伸"
	case sig.Anxious && i == 1:
		return "4-7-8 呼吸安定"
	case sig.Angry && i == 4:
		return "安全地释放张力"
	default:
		return base[i]
	}
}

func dayTasks(sig PlanSignal, i int) []RecTask {
	breathe := RecTask{Slot: constants.PlanSlotBreathe, Title: "呼吸练习 3 分钟", Content: "缓慢吸气 4 秒、屏息 4 秒、呼气 6 秒，重复 10 轮，注意腹部的起伏。"}
	move := RecTask{Slot: constants.PlanSlotMove, Title: "户外散步 15 分钟", Content: "到有自然光的地方慢走，留意脚步与周围声音，不追求强度。"}
	wind := RecTask{Slot: constants.PlanSlotWind, Title: "睡前放下手机 30 分钟", Content: "调暗灯光，做简单拉伸或听舒缓音频，给入睡留出过渡。"}
	connect := RecTask{Slot: constants.PlanSlotConnect, Title: "与信任的人简短联系", Content: "给一位朋友发一条问候，或写下一句想对他人说的话。"}

	switch {
	case sig.Anxious:
		breathe = RecTask{Slot: constants.PlanSlotBreathe, Title: "4-7-8 着陆呼吸", Content: "吸气 4 秒、屏息 7 秒、呼气 8 秒，重复 4 轮；再说出眼前看到的 5 样东西。"}
	case sig.Angry:
		breathe = RecTask{Slot: constants.PlanSlotBreathe, Title: "降温呼吸 5 分钟", Content: "先用冷水洗手腕，再做缓慢深呼吸，给情绪一个缓冲空间。"}
	}
	switch {
	case sig.Tired:
		move = RecTask{Slot: constants.PlanSlotMove, Title: "恢复性拉伸 10 分钟", Content: "轻柔伸展颈肩与腰背，每个动作保持自然呼吸，不勉强。"}
	case sig.MoodLow:
		move = RecTask{Slot: constants.PlanSlotMove, Title: "走到窗边晒晒太阳", Content: "不设定距离目标，只需起身、接触光线并喝一杯温水。"}
	case sig.Angry:
		move = RecTask{Slot: constants.PlanSlotMove, Title: "稍快步行 20 分钟", Content: "用有节奏的步伐释放张力，结束后做三次缓慢呼气。"}
	}
	if sig.Tired || sig.StressHigh {
		wind = RecTask{Slot: constants.PlanSlotWind, Title: "固定就寝时间", Content: "今晚比平时提前 20 分钟上床，睡前写下明天的一件小事即可。"}
	}

	tasks := []RecTask{breathe, move}
	// 第 3 天和第 6 天用“人际连接/回顾”替换睡前项，形成七日节奏。
	if i == 2 {
		tasks = append(tasks, connect)
	} else if i == 5 {
		journal := RecTask{Slot: constants.PlanSlotWind, Title: "给这一周的自己写三句话", Content: "写下：这周辛苦的一件事、感激的一件事、下周想温柔对待自己的一件事。"}
		tasks = append(tasks, journal)
	} else {
		tasks = append(tasks, wind)
	}
	if i == 6 && sig.StressHigh {
		tasks = append(tasks, RecTask{Slot: constants.PlanSlotConnect, Title: "了解专业支持渠道", Content: "若困扰持续影响生活，记下心理咨询热线或专业机构，求助本身就是力量。"})
	}
	return tasks
}

func parseMoodTags(raw string) []string {
	var tags []string
	if raw == "" {
		return tags
	}
	if e := json.Unmarshal([]byte(raw), &tags); e != nil {
		return nil
	}
	return tags
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func truncateDay(t time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.UTC
	}
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}
