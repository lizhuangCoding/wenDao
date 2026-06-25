export type SelectedChatModel = { provider: string; model_name: string };

const ACTIVE_CHAT_STORAGE_KEY = 'wendao.aiChat.activeId';
const MODEL_STORAGE_KEY = 'wendao.aiChat.model';

export const readStoredActiveId = () => {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(ACTIVE_CHAT_STORAGE_KEY);
  if (!raw) return null;
  const parsed = Number(raw);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
};

export const readStoredModel = (): SelectedChatModel | null => {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(MODEL_STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed.provider === 'string' && typeof parsed.model_name === 'string') {
      return parsed as SelectedChatModel;
    }
  } catch {
    // ignore
  }
  return null;
};

export const persistActiveChatId = (id: number | null) => {
  if (typeof window === 'undefined') return;
  if (id === null) {
    window.localStorage.removeItem(ACTIVE_CHAT_STORAGE_KEY);
    return;
  }
  window.localStorage.setItem(ACTIVE_CHAT_STORAGE_KEY, String(id));
};

export const persistSelectedModel = (model: SelectedChatModel | null) => {
  if (typeof window === 'undefined') return;
  if (model) {
    window.localStorage.setItem(MODEL_STORAGE_KEY, JSON.stringify(model));
    return;
  }
  window.localStorage.removeItem(MODEL_STORAGE_KEY);
};
