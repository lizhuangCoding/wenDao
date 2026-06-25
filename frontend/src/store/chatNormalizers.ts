import type { ChatActiveRun, ChatConversationDetailResponse, ChatMessage, ChatStep, ChatStepEvent } from '@/types';

export interface Conversation {
  id: number;
  title: string;
  messages: ChatMessage[];
  steps: ChatStep[];
  activeRun: ChatActiveRun | null;
  createdAt: number;
  updatedAt: number;
  isLoaded: boolean;
  isShared?: boolean;
  shareToken?: string;
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null;
};

const readString = (record: Record<string, unknown>, key: string, fallback = '') => {
  const value = record[key];
  return typeof value === 'string' ? value : fallback;
};

const readNumber = (record: Record<string, unknown>, key: string) => {
  const value = record[key];
  if (typeof value === 'number') return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
};

const readArray = (record: Record<string, unknown>, key: string): unknown[] | undefined => {
  const value = record[key];
  return Array.isArray(value) ? value : undefined;
};

const readChatRole = (record: Record<string, unknown>): ChatMessage['role'] => {
  return record.role === 'assistant' ? 'assistant' : 'user';
};

const readStepStatus = (record: Record<string, unknown>): ChatStep['status'] => {
  const status = record.status;
  return status === 'completed' || status === 'failed' ? status : 'running';
};

const normalizeStep = (step: unknown): ChatStep => {
  const record = isRecord(step) ? step : {};
  return {
    id: Number(readNumber(record, 'id') ?? readNumber(record, 'step_id') ?? 0),
    run_id: readNumber(record, 'run_id'),
    agent_name: readString(record, 'agent_name', readString(record, 'agentName', 'Agent')),
    type: readString(record, 'type', 'thinking'),
    summary: readString(record, 'summary'),
    detail: readString(record, 'detail'),
    status: readStepStatus(record),
    created_at: readString(record, 'created_at', new Date().toISOString()),
  };
};

const mapSteps = (steps: unknown[] = []): ChatStep[] => steps.map(normalizeStep);

const parseChatTime = (value?: string) => {
  if (!value) return 0;
  const normalized = value.includes('T') ? value : value.replace(' ', 'T');
  const time = new Date(normalized).getTime();
  return Number.isFinite(time) ? time : 0;
};

const attachStepsToLatestAssistant = (messages: ChatMessage[], steps: ChatStep[]): ChatMessage[] => {
  if (steps.length === 0) return messages;
  const latestAssistantIndex = [...messages].reverse().findIndex((m) => m.role === 'assistant');
  if (latestAssistantIndex === -1) return messages;
  const targetIndex = messages.length - 1 - latestAssistantIndex;
  return messages.map((message, index) =>
    index === targetIndex ? { ...message, processSteps: steps } : message
  );
};

type StepGroup = {
  runId: number;
  steps: ChatStep[];
  firstStepAt: number;
  lastStepAt: number;
};

const groupStepsByRun = (steps: ChatStep[]): StepGroup[] => {
  const groups = new Map<number, ChatStep[]>();
  for (const step of steps) {
    if (!step.run_id) continue;
    const existing = groups.get(step.run_id) || [];
    existing.push(step);
    groups.set(step.run_id, existing);
  }
  return Array.from(groups.entries())
    .map(([runId, runSteps]) => {
      const times = runSteps.map((step) => parseChatTime(step.created_at)).filter((time) => time > 0);
      return {
        runId,
        steps: runSteps,
        firstStepAt: times.length ? Math.min(...times) : 0,
        lastStepAt: times.length ? Math.max(...times) : 0,
      };
    })
    .sort((a, b) => a.firstStepAt - b.firstStepAt);
};

const attachStepsToAssistantMessages = (messages: ChatMessage[], steps: ChatStep[]): ChatMessage[] => {
  if (steps.length === 0) return messages;

  const assistantIndexes = messages.reduce<number[]>((indexes, message, index) => {
    if (message.role === 'assistant') indexes.push(index);
    return indexes;
  }, []);
  if (assistantIndexes.length === 0) return messages;

  const groupedSteps = groupStepsByRun(steps);
  if (groupedSteps.length === 0) {
    return attachStepsToLatestAssistant(messages, steps);
  }

  const stepsByMessageIndex = new Map<number, ChatStep[]>();
  const assignedMessageIndexes = new Set<number>();
  const unassignedGroups: StepGroup[] = [];

  groupedSteps.forEach((group) => {
    const directIndex = assistantIndexes.find((messageIndex) => {
      if (assignedMessageIndexes.has(messageIndex)) return false;
      return messages[messageIndex].runId === group.runId;
    });
    if (directIndex !== undefined) {
      assignedMessageIndexes.add(directIndex);
      stepsByMessageIndex.set(directIndex, group.steps);
      return;
    }
    unassignedGroups.push(group);
  });

  unassignedGroups.forEach((group) => {
    const candidateIndex = assistantIndexes.find((messageIndex) => {
      if (assignedMessageIndexes.has(messageIndex)) return false;
      const message = messages[messageIndex];
      return group.lastStepAt > 0 && message.timestamp >= group.lastStepAt;
    });

    const fallbackIndex = assistantIndexes.find((messageIndex) => !assignedMessageIndexes.has(messageIndex));
    const messageIndex = candidateIndex ?? fallbackIndex;
    if (messageIndex === undefined) return;

    assignedMessageIndexes.add(messageIndex);
    stepsByMessageIndex.set(messageIndex, group.steps);
  });

  return messages.map((message, index) => {
    const processSteps = stepsByMessageIndex.get(index);
    if (!processSteps) return message;
    return {
      ...message,
      processSteps,
      runId: processSteps[0]?.run_id,
    };
  });
};

const mapMessages = (messages: unknown[] = [], steps: unknown[] = []): ChatMessage[] => {
  const mapped = messages.map((message) => {
    const record = isRecord(message) ? message : {};
    const processStepValues = readArray(record, 'process_steps');
    return {
      id: String(record.id ?? ''),
      role: readChatRole(record),
      content: readString(record, 'content'),
      timestamp: parseChatTime(readString(record, 'created_at')),
      runId: readNumber(record, 'run_id'),
      processSteps: processStepValues ? mapSteps(processStepValues) : undefined,
    };
  });
  return attachStepsToAssistantMessages(mapped, mapSteps(steps));
};

export const stepEventToStep = (event: ChatStepEvent): ChatStep => ({
  id: Number(event.step_id || 0),
  run_id: event.run_id ? Number(event.run_id) : undefined,
  agent_name: event.agent_name || 'Agent',
  type: 'thinking',
  summary: event.summary || '',
  detail: event.detail || '',
  status: event.status || 'running',
  created_at: new Date().toISOString(),
});

export const upsertStep = (steps: ChatStep[] = [], next: ChatStep): ChatStep[] => {
  const key = next.id > 0 ? `id:${next.id}` : `agent:${next.agent_name}:${next.summary}`;
  const index = steps.findIndex((step) => {
    const stepKey = step.id > 0 ? `id:${step.id}` : `agent:${step.agent_name}:${step.summary}`;
    return stepKey === key;
  });
  if (index === -1) return [...steps, next];
  return steps.map((step, i) => (i === index ? { ...step, ...next } : step));
};

const mapActiveRun = (run?: ChatConversationDetailResponse['active_run']): ChatActiveRun | null => {
  if (!run) return null;
  if (!run.can_resume || (run.status !== 'running' && run.status !== 'waiting_user')) return null;
  return {
    id: Number(run.id),
    status: run.status,
    current_stage: run.current_stage,
    pending_question: run.pending_question,
    last_answer: run.last_answer ?? '',
    heartbeat_at: run.heartbeat_at,
    can_resume: Boolean(run.can_resume),
  };
};

export const ensureResumableAssistantMessage = (
  messages: ChatMessage[],
  activeRun: ChatActiveRun | null,
  activeSteps: ChatStep[] = []
): ChatMessage[] => {
  if (!activeRun) return messages;
  const content = activeRun.pending_question ?? activeRun.last_answer ?? '';
  const targetIndex = messages.findIndex(
    (message) => message.role === 'assistant' && (message.runId === activeRun.id || message.id === `resume-${activeRun.id}`)
  );
  if (targetIndex !== -1) {
    return messages.map((message, index) =>
      index === targetIndex
        ? {
            ...message,
            content: content || message.content,
            processSteps: activeSteps.length ? activeSteps : message.processSteps,
            runId: activeRun.id,
          }
        : message
    );
  }
  return [
    ...messages,
    {
      id: `resume-${activeRun.id}`,
      role: 'assistant',
      content,
      timestamp: Date.now(),
      processSteps: activeSteps,
      runId: activeRun.id,
    },
  ];
};

export const mapConversationDetail = (detail: ChatConversationDetailResponse): Conversation => {
  const steps = mapSteps(detail.steps ?? []);
  const activeSteps = mapSteps(detail.active_steps ?? []);
  const activeRun = mapActiveRun(detail.active_run);
  const historicalSteps = activeRun ? steps.filter((step) => step.run_id !== activeRun.id) : steps;
  const messages = ensureResumableAssistantMessage(mapMessages(detail.messages, historicalSteps), activeRun, activeSteps);

  return {
    id: detail.conversation.id,
    title: detail.conversation.title,
    messages,
    steps: activeSteps.length ? activeSteps : steps,
    activeRun,
    createdAt: new Date(detail.conversation.created_at).getTime(),
    updatedAt: new Date(detail.conversation.updated_at).getTime(),
    isLoaded: true,
    isShared: detail.conversation.is_shared,
    shareToken: detail.conversation.share_token,
  };
};

export const preserveExistingProcessSteps = (next: Conversation, previous?: Conversation): Conversation => {
  if (!previous) return next;

  const previousAssistantMessages = previous.messages.filter((message) => message.role === 'assistant');
  const stepsByRunId = new Map<number, { steps: ChatStep[]; runId?: number }>();
  const stepsByAssistantIndex: Array<{ steps: ChatStep[]; runId?: number }> = [];

  previousAssistantMessages.forEach((message) => {
    if (!message.processSteps?.length) return;
    const entry = { steps: message.processSteps, runId: message.runId };
    stepsByAssistantIndex.push(entry);
    if (message.runId) {
      stepsByRunId.set(message.runId, entry);
    }
  });

  if (stepsByRunId.size === 0 && stepsByAssistantIndex.length === 0) return next;

  let assistantIndex = 0;
  const messages = next.messages.map((message) => {
    if (message.role !== 'assistant') return message;

    const currentAssistantIndex = assistantIndex;
    assistantIndex += 1;

    if (message.processSteps?.length) return message;

    const matched = message.runId ? stepsByRunId.get(message.runId) : undefined;
    const fallback = stepsByAssistantIndex[currentAssistantIndex];
    const preserved = matched ?? fallback;
    if (!preserved?.steps.length) return message;

    return {
      ...message,
      processSteps: preserved.steps,
      runId: message.runId ?? preserved.runId,
    };
  });

  return { ...next, messages };
};
