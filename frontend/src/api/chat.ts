import { getApiUrl, request } from './client';
import type {
  ChatHeartbeatEvent,
  ChatConversationDetailResponse,
  ChatQuestionEvent,
  ChatRequest,
  ChatResponse,
  ChatResumeEvent,
  ChatSnapshotEvent,
  ChatStageEvent,
  ChatStepEvent,
  ModelInfo,
  SharedConversationData,
} from '@/types';

type ChatStreamHandlers = {
  onStart?: (payload: Record<string, unknown>) => void;
  onResume?: (payload: ChatResumeEvent) => void;
  onSnapshot?: (payload: ChatSnapshotEvent) => void;
  onHeartbeat?: (payload: ChatHeartbeatEvent) => void;
  onStage?: (payload: ChatStageEvent) => void;
  onQuestion?: (payload: ChatQuestionEvent) => void;
  onStep?: (payload: ChatStepEvent) => void;
  onChunk: (payload: { run_id?: number; message?: string; content?: string; sources?: string[] }) => void;
  onDone?: (payload: Record<string, unknown>) => void;
  onError?: (payload: { error?: string; message?: string }) => void;
};

async function readSSEStream(response: Response, handlers: ChatStreamHandlers) {
  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }

  const decoder = new TextDecoder('utf-8');
  let buffer = '';

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split('\n\n');
    buffer = events.pop() || '';

    for (const rawEvent of events) {
      const lines = rawEvent.split('\n');
      const eventLine = lines.find((line) => line.startsWith('event: '));
      const dataLine = lines.find((line) => line.startsWith('data: '));
      if (!eventLine || !dataLine) continue;

      const eventName = eventLine.replace('event: ', '').trim();
      const payload = JSON.parse(dataLine.replace('data: ', '').trim());

      if (eventName === 'start') handlers.onStart?.(payload);
      if (eventName === 'resume') handlers.onResume?.(payload);
      if (eventName === 'snapshot') handlers.onSnapshot?.(payload);
      if (eventName === 'heartbeat') handlers.onHeartbeat?.(payload);
      if (eventName === 'stage') handlers.onStage?.(payload);
      if (eventName === 'question') handlers.onQuestion?.(payload);
      if (eventName === 'step') handlers.onStep?.(payload);
      if (eventName === 'chunk') handlers.onChunk(payload);
      if (eventName === 'done') handlers.onDone?.(payload);
      if (eventName === 'error') handlers.onError?.(payload);
    }
  }
}

const sanitizeConversationExportTitle = (title: string) => {
  const safeTitle = title.trim().replace(/[\\/:*?"<>|]/g, '_');
  return safeTitle || 'conversation';
};

export const chatApi = {
  sendMessage: (data: ChatRequest) => {
    return request.post<ChatResponse>('/ai/chat', data);
  },

  streamMessage: async (data: ChatRequest, handlers: ChatStreamHandlers) => {
    const token = localStorage.getItem('access_token');
    const response = await fetch(getApiUrl('/ai/chat/stream'), {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error('流式请求失败');
    }

    await readSSEStream(response, handlers);
  },

  resumeStream: async (conversationId: number, runId: number, handlers: ChatStreamHandlers) => {
    const token = localStorage.getItem('access_token');
    const response = await fetch(getApiUrl('/ai/chat/stream/resume'), {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ conversation_id: conversationId, run_id: runId }),
    });

    if (!response.ok) {
      throw new Error('恢复流式请求失败');
    }

    await readSSEStream(response, handlers);
  },

  getConversations: () => {
    return request.get<any[]>('/chat/conversations');
  },

  createConversation: (title: string) => {
    return request.post<{ id: number; title: string; user_id: number; created_at: string; updated_at: string }>('/chat/conversations', { title });
  },

  getConversation: (id: number) => {
    return request.get<ChatConversationDetailResponse>(`/chat/conversations/${id}`);
  },

  renameConversation: (id: number, title: string) => {
    return request.patch<{ id: number; title: string; user_id: number; created_at: string; updated_at: string }>(`/chat/conversations/${id}`, { title });
  },

  deleteConversation: (id: number) => {
    return request.delete<any>(`/chat/conversations/${id}`);
  },

  generateSummary: (content: string) => {
    return request.post<{ summary: string }>('/ai/summary', { content });
  },

  getModels: () => {
    return request.get<{ models: ModelInfo[] }>('/models');
  },

  shareConversation: (id: number, share: boolean) => {
    return request.post<{ id: number; is_shared: boolean; share_token: string }>(`/chat/conversations/${id}/share`, { share });
  },

  getSharedConversation: (token: string) => {
    return request.get<SharedConversationData>(`/shared/conversations/${token}`);
  },

  exportConversation: async (id: number, title: string) => {
    const token = localStorage.getItem('access_token');
    const response = await fetch(getApiUrl(`/chat/conversations/${id}/export`), {
      credentials: 'include',
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
    if (!response.ok) {
      throw new Error('导出失败');
    }
    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    const safeTitle = sanitizeConversationExportTitle(title);
    a.href = url;
    a.download = `conversation-${safeTitle}.md`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },
};
