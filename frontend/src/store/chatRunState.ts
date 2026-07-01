import type { ChatStage } from '@/types';

export type ChatRunStatus = 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';

export interface ChatRunState {
  isTyping: boolean;
  isStreaming: boolean;
  streamingConversationId: number | null;
  currentStage: ChatStage | null;
  currentStageLabel: string | null;
  requiresUserInput: boolean;
  pendingQuestion: string | null;
  runStatus: ChatRunStatus;
  isRecovering: boolean;
  reconnectAttempts: number;
  lastHeartbeatAt: number | null;
}

export const createIdleChatRunState = (): ChatRunState => ({
  isTyping: false,
  isStreaming: false,
  streamingConversationId: null,
  currentStage: null,
  currentStageLabel: null,
  requiresUserInput: false,
  pendingQuestion: null,
  runStatus: 'idle',
  isRecovering: false,
  reconnectAttempts: 0,
  lastHeartbeatAt: null,
});

export const createRecoveringChatRunState = ({
  conversationId,
  currentStage,
  status,
  reconnectAttempts,
  pendingQuestion = null,
}: {
  conversationId: number;
  currentStage: ChatStage | null;
  status: ChatRunStatus;
  reconnectAttempts: number;
  pendingQuestion?: string | null;
}): ChatRunState => ({
  isTyping: status === 'running',
  isStreaming: status === 'running',
  streamingConversationId: status === 'running' ? conversationId : null,
  currentStage,
  currentStageLabel: status === 'waiting_user' ? '需要你补充一点信息' : '正在恢复回答',
  requiresUserInput: status === 'waiting_user',
  pendingQuestion,
  runStatus: status,
  isRecovering: true,
  reconnectAttempts,
  lastHeartbeatAt: Date.now(),
});

export const createWaitingUserChatRunState = (pendingQuestion: string): ChatRunState => ({
  isTyping: false,
  isStreaming: false,
  streamingConversationId: null,
  currentStage: 'clarifying',
  currentStageLabel: '需要你补充一点信息',
  requiresUserInput: true,
  pendingQuestion,
  runStatus: 'waiting_user',
  isRecovering: false,
  reconnectAttempts: 0,
  lastHeartbeatAt: Date.now(),
});

export const createCompletedChatRunState = (): ChatRunState => ({
  isTyping: false,
  isStreaming: false,
  streamingConversationId: null,
  currentStage: 'completed',
  currentStageLabel: '回答已生成',
  requiresUserInput: false,
  pendingQuestion: null,
  runStatus: 'completed',
  isRecovering: false,
  reconnectAttempts: 0,
  lastHeartbeatAt: null,
});

export const createFailedChatRunState = (pendingQuestion: string | null = null): ChatRunState => ({
  isTyping: false,
  isStreaming: false,
  streamingConversationId: null,
  currentStage: null,
  currentStageLabel: null,
  requiresUserInput: false,
  pendingQuestion,
  runStatus: 'failed',
  isRecovering: false,
  reconnectAttempts: 0,
  lastHeartbeatAt: null,
});
