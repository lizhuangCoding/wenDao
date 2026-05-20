import type { ChatStage, ChatStep } from '@/types';

export type AgentMoodKey =
  | 'clarifier'
  | 'executor'
  | 'found'
  | 'journalist'
  | 'librarian'
  | 'planner'
  | 'replanner'
  | 'reviewer'
  | 'synthesizer'
  | 'thinking';

export interface AgentMoodInput {
  agentName?: string | null;
  stage?: ChatStage | 'streaming' | null;
  status?: ChatStep['status'];
  summary?: string | null;
}

export interface AgentMood {
  caption: string;
  key: AgentMoodKey;
  label: string;
  tone: 'amber' | 'blue' | 'cyan' | 'emerald' | 'fuchsia' | 'indigo' | 'rose' | 'violet';
}

const normalize = (value?: string | null) => (value || '').trim().toLowerCase();

const hasMatchedAnswerSignal = (summary?: string | null) => {
  const normalized = normalize(summary);
  return /匹配|命中|充足|可直接|sufficient|matched|found/.test(normalized);
};

export const resolveAgentMood = ({
  agentName,
  stage,
  status,
  summary,
}: AgentMoodInput): AgentMood => {
  if (status === 'completed' && hasMatchedAnswerSignal(summary)) {
    return {
      caption: '找到了值得展开的线索',
      key: 'found',
      label: '发现高匹配答案',
      tone: 'amber',
    };
  }

  const agent = normalize(agentName);
  const currentStage = normalize(stage);
  const combined = `${agent} ${currentStage}`;

  if (combined.includes('librarian') || combined.includes('local_search')) {
    return {
      caption: '戴上眼镜翻找站内知识',
      key: 'librarian',
      label: 'Librarian 正在查图书馆',
      tone: 'emerald',
    };
  }

  if (combined.includes('journalist') || combined.includes('web_research')) {
    return {
      caption: '搜索外部线索并校验来源',
      key: 'journalist',
      label: 'Journalist 正在外部调研',
      tone: 'cyan',
    };
  }

  if (combined.includes('synthesizer') || combined.includes('synthesizing') || combined.includes('integration')) {
    return {
      caption: '把线索编织成完整回答',
      key: 'synthesizer',
      label: 'Synthesizer 正在整合观点',
      tone: 'violet',
    };
  }

  if (combined.includes('reviewer') || combined.includes('reviewing') || combined.includes('revising')) {
    return {
      caption: '检查结论、漏洞和表达质量',
      key: 'reviewer',
      label: 'Reviewer 正在校验答案',
      tone: 'rose',
    };
  }

  if (combined.includes('replanner')) {
    return {
      caption: '根据新结果调整路线',
      key: 'replanner',
      label: 'Replanner 正在重规划',
      tone: 'fuchsia',
    };
  }

  if (combined.includes('planner') || combined.includes('analyzing')) {
    return {
      caption: '拆解任务并规划路径',
      key: 'planner',
      label: 'Planner 正在规划路线',
      tone: 'blue',
    };
  }

  if (combined.includes('executor')) {
    return {
      caption: '执行计划中的当前步骤',
      key: 'executor',
      label: 'Executor 正在行动',
      tone: 'indigo',
    };
  }

  if (combined.includes('clarifying')) {
    return {
      caption: '确认问题边界和目标',
      key: 'clarifier',
      label: 'Clarifier 正在追问关键点',
      tone: 'amber',
    };
  }

  return {
    caption: '保持思考，等待下一条线索',
    key: 'thinking',
    label: 'AI 助手正在思考',
    tone: 'emerald',
  };
};
