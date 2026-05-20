export type ChatQuestionNavRole = 'user' | 'assistant';

export interface ChatQuestionNavMessage {
  id: string;
  role: ChatQuestionNavRole;
  content: string;
  timestamp?: number;
}

export interface ChatQuestionNavItem {
  anchorId: string;
  fullText: string;
  index: number;
  label: string;
  messageId: string;
  timestamp?: number;
}

const DEFAULT_LABEL_LENGTH = 64;

const normalizeQuestionText = (content: string) => content.trim().replace(/\s+/g, ' ');

const sanitizeAnchorPart = (value: string) => {
  const sanitized = value.trim().replace(/[^a-zA-Z0-9_-]+/g, '-').replace(/^-+|-+$/g, '');
  return sanitized || 'message';
};

export const getChatQuestionAnchorId = (messageId: string, index: number) =>
  `chat-question-${index}-${sanitizeAnchorPart(messageId)}`;

const shortenLabel = (text: string, maxLength: number) => {
  if (text.length <= maxLength) return text;
  return `${text.slice(0, Math.max(0, maxLength))}...`;
};

export const buildChatQuestionNavItems = (
  messages: ChatQuestionNavMessage[],
  maxLabelLength = DEFAULT_LABEL_LENGTH
): ChatQuestionNavItem[] => {
  let questionIndex = 0;

  return messages.reduce<ChatQuestionNavItem[]>((items, message) => {
    if (message.role !== 'user') return items;

    const fullText = normalizeQuestionText(message.content);
    if (!fullText) return items;

    questionIndex += 1;
    items.push({
      anchorId: getChatQuestionAnchorId(message.id, questionIndex),
      fullText,
      index: questionIndex,
      label: shortenLabel(fullText, maxLabelLength),
      messageId: message.id,
      timestamp: message.timestamp,
    });

    return items;
  }, []);
};
