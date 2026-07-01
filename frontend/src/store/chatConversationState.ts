import type {
  ChatConversationMutationResponse,
  ChatConversationSummary,
} from '@/types';
import type { Conversation } from './chatNormalizers';

export const createConversationRecord = (
  response: Pick<ChatConversationMutationResponse, 'id' | 'title' | 'created_at' | 'updated_at'>
): Conversation => ({
  id: response.id,
  title: response.title,
  messages: [],
  steps: [],
  activeRun: null,
  createdAt: new Date(response.created_at).getTime(),
  updatedAt: new Date(response.updated_at).getTime(),
  isLoaded: true,
});

export const createConversationSummaryRecord = (
  conversation: ChatConversationSummary
): Conversation => ({
  id: conversation.id,
  title: conversation.title,
  messages: [],
  steps: [],
  activeRun: null,
  createdAt: new Date(conversation.created_at).getTime(),
  updatedAt: new Date(conversation.updated_at).getTime(),
  isLoaded: false,
  isShared: conversation.is_shared,
  shareToken: conversation.share_token,
});

export const createConversationMap = (
  conversations: ChatConversationSummary[]
): Record<number, Conversation> => {
  const records: Record<number, Conversation> = {};
  for (const conversation of conversations) {
    records[conversation.id] = createConversationSummaryRecord(conversation);
  }
  return records;
};

export const removeConversationRecord = <T extends Record<number, Conversation>>(conversations: T, id: number) => {
  const next = { ...conversations };
  delete next[id];
  return next;
};

export const updateConversationShareRecord = (
  conversation: Conversation,
  isShared: boolean,
  shareToken?: string
): Conversation => ({
  ...conversation,
  isShared,
  shareToken,
});
