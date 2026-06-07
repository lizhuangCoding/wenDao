import type { ChatStage, ChatStep } from '@/types';
import i18n from '@/i18n';

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
  detail?: string | null;
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
const isEnglish = () => (i18n.resolvedLanguage || i18n.language || 'zh').startsWith('en');

const text = (zh: string, en: string) => (isEnglish() ? en : zh);

const hasMatchedAnswerSignal = (summary?: string | null, detail?: string | null) => {
  const normalized = normalize(`${summary || ''}\n${detail || ''}`);
  const hasExplicitCoverage = /覆盖状态[:：]\s*sufficient|coverage_status["']?\s*[:=]\s*["']?sufficient\b/.test(normalized);
  const hasPositiveSignal = /匹配|命中|资料充足|站内资料充足|可直接回答|\bsufficient\b|\bmatched\b|\bfound\b/.test(normalized);
  const hasNegativeSignal = /不匹配|未命中|无匹配|资料不足|站内资料不足|不充足|未覆盖|没有覆盖|没有关于|\binsufficient\b/.test(normalized);

  return hasExplicitCoverage || (hasPositiveSignal && !hasNegativeSignal);
};

export const isMatchedAnswerStep = (step: ChatStep) => {
  return step.status === 'completed' && hasMatchedAnswerSignal(step.summary, step.detail);
};

export const selectFeaturedAgentStep = (steps: ChatStep[]) => {
  for (let index = steps.length - 1; index >= 0; index -= 1) {
    if (isMatchedAnswerStep(steps[index])) {
      return steps[index];
    }
  }

  for (let index = steps.length - 1; index >= 0; index -= 1) {
    if (steps[index].status === 'running') {
      return steps[index];
    }
  }

  return steps[steps.length - 1] || null;
};

export const resolveAgentMood = ({
  agentName,
  detail,
  stage,
  status,
  summary,
}: AgentMoodInput): AgentMood => {
  if (status === 'completed' && hasMatchedAnswerSignal(summary, detail)) {
    return {
      caption: text('找到了值得展开的线索', 'Found a lead worth expanding'),
      key: 'found',
      label: text('发现高匹配答案', 'High-match answer found'),
      tone: 'amber',
    };
  }

  const agent = normalize(agentName);
  const currentStage = normalize(stage);
  const combined = `${agent} ${currentStage}`;

  if (combined.includes('librarian') || combined.includes('local_search')) {
    return {
      caption: text('戴上眼镜翻找站内知识', 'Searching the internal knowledge base'),
      key: 'librarian',
      label: text('Librarian 正在查图书馆', 'Librarian is searching the library'),
      tone: 'emerald',
    };
  }

  if (combined.includes('journalist') || combined.includes('web_research')) {
    return {
      caption: text('搜索外部线索并校验来源', 'Searching external leads and verifying sources'),
      key: 'journalist',
      label: text('Journalist 正在外部调研', 'Journalist is researching externally'),
      tone: 'cyan',
    };
  }

  if (combined.includes('synthesizer') || combined.includes('synthesizing') || combined.includes('integration')) {
    return {
      caption: text('把线索编织成完整回答', 'Weaving signals into a complete answer'),
      key: 'synthesizer',
      label: text('Synthesizer 正在整合观点', 'Synthesizer is integrating perspectives'),
      tone: 'violet',
    };
  }

  if (combined.includes('reviewer') || combined.includes('reviewing') || combined.includes('revising')) {
    return {
      caption: text('检查结论、漏洞和表达质量', 'Checking conclusions, gaps, and clarity'),
      key: 'reviewer',
      label: text('Reviewer 正在校验答案', 'Reviewer is validating the answer'),
      tone: 'rose',
    };
  }

  if (combined.includes('replanner')) {
    return {
      caption: text('根据新结果调整路线', 'Adjusting the route based on new results'),
      key: 'replanner',
      label: text('Replanner 正在重规划', 'Replanner is updating the plan'),
      tone: 'fuchsia',
    };
  }

  if (combined.includes('planner') || combined.includes('analyzing')) {
    return {
      caption: text('拆解任务并规划路径', 'Breaking down the task and planning a path'),
      key: 'planner',
      label: text('Planner 正在规划路线', 'Planner is mapping the route'),
      tone: 'blue',
    };
  }

  if (combined.includes('executor')) {
    return {
      caption: text('执行计划中的当前步骤', 'Executing the current step'),
      key: 'executor',
      label: text('Executor 正在行动', 'Executor is taking action'),
      tone: 'indigo',
    };
  }

  if (combined.includes('clarifying')) {
    return {
      caption: text('确认问题边界和目标', 'Clarifying the scope and goal'),
      key: 'clarifier',
      label: text('Clarifier 正在追问关键点', 'Clarifier is asking for key details'),
      tone: 'amber',
    };
  }

  return {
    caption: text('保持思考，等待下一条线索', 'Thinking and waiting for the next signal'),
    key: 'thinking',
    label: text('AI 助手正在思考', 'AI Assistant is thinking'),
    tone: 'emerald',
  };
};
