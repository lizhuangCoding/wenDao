import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { chatApi } from '@/api';
import { Layout } from '@/components/common';
import { ChatMessageList } from '@/components/chat/ChatMessageList';
import { ChatQuestionNavigator } from '@/components/chat/ChatQuestionNavigator';
import { useMemo, useCallback, useRef } from 'react';
import { buildChatQuestionNavItems } from '@/utils/chatQuestionNavigator';
import type { SharedConversationData, ChatMessage } from '@/types';

const emptyMessages: ChatMessage[] = [];

export const SharedConversation = () => {
  const { t, i18n } = useTranslation();
  const { token } = useParams<{ token: string }>();
  const [data, setData] = useState<SharedConversationData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [expandedProcessIds, setExpandedProcessIds] = useState<Set<string>>(new Set());
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!token) {
      setError(t('sharedConversation.invalidLink'));
      setLoading(false);
      return;
    }

    chatApi.getSharedConversation(token).then((res) => {
      setData(res);
    }).catch(() => {
      setError(t('sharedConversation.conversationNotFound'));
    }).finally(() => {
      setLoading(false);
    });
  }, [t, token]);

  const messages: ChatMessage[] = useMemo(() => {
    if (!data) return emptyMessages;
    return data.messages.map((msg) => ({
      id: String(msg.id),
      role: msg.role,
      content: msg.content,
      timestamp: new Date(msg.created_at).getTime(),
      processSteps: msg.process_steps || [],
      runId: msg.run_id,
    }));
  }, [data]);

  const questionNavItems = useMemo(() => buildChatQuestionNavItems(messages), [messages]);
  const questionAnchorByMessageId = useMemo(
    () => new Map(questionNavItems.map((item) => [item.messageId, item.anchorId])),
    [questionNavItems]
  );

  const scrollToQuestion = useCallback((anchorId: string) => {
    const container = scrollContainerRef.current;
    const element = document.getElementById(anchorId);
    if (!container || !element) return;
    container.scrollTo({ top: Math.max(element.offsetTop - 24, 0), behavior: 'smooth' });
  }, []);

  const toggleProcessDetail = (id: string) => {
    setExpandedProcessIds((current) => {
      const next = new Set(current);
      if (next.has(id)) { next.delete(id); } else { next.add(id); }
      return next;
    });
  };

  if (loading) {
    return (
      <Layout>
        <div className="max-w-4xl mx-auto px-4 py-20 text-center">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-neutral-100 dark:bg-neutral-800 rounded w-48 mx-auto" />
            <div className="h-4 bg-neutral-100 dark:bg-neutral-800 rounded w-64 mx-auto" />
          </div>
        </div>
      </Layout>
    );
  }

  if (error || !data) {
    return (
      <Layout>
        <div className="max-w-4xl mx-auto px-4 py-20 text-center">
          <h1 className="text-2xl font-serif font-black text-neutral-900 dark:text-neutral-100 mb-4">
            {t('sharedConversation.failedToLoad')}
          </h1>
          <p className="text-neutral-500 dark:text-neutral-400 mb-8">
            {error || t('sharedConversation.unknownError')}
          </p>
          <a
            href="/"
            className="inline-flex items-center gap-2 rounded-xl bg-primary-600 px-6 py-3 text-sm font-bold text-white hover:bg-primary-700 transition-colors"
          >
            {t('sharedConversation.returnHome')}
          </a>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="max-w-4xl mx-auto px-3 py-3 sm:px-8 sm:py-6 lg:px-10 lg:py-10">
        {/* Header */}
        <div className="mb-6 sm:mb-8">
          <div className="flex items-center gap-3 mb-2">
            {data.shared_by.avatar_url && (
              <img
                src={data.shared_by.avatar_url}
                alt={data.shared_by.username}
                className="h-8 w-8 rounded-full"
              />
            )}
            <span className="text-sm font-bold text-neutral-600 dark:text-neutral-400">
              {t('sharedConversation.sharedBy', { username: data.shared_by.username })}
            </span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-serif font-black text-neutral-900 dark:text-neutral-100">
            {data.conversation.title}
          </h1>
          <p className="mt-2 text-xs text-neutral-400 dark:text-neutral-500">
            {t('sharedConversation.createdAt', {
              date: new Date(data.conversation.created_at).toLocaleDateString(
                i18n.resolvedLanguage?.startsWith('en') ? 'en-US' : 'zh-CN'
              ),
            })}
          </p>
        </div>

        <div className="bg-white dark:bg-neutral-800 rounded-2xl sm:rounded-[32px] border border-neutral-100 dark:border-neutral-700 shadow-soft overflow-hidden">
          <ChatQuestionNavigator
            activeId={null}
            items={questionNavItems}
            onSelect={scrollToQuestion}
          />

          <div
            ref={scrollContainerRef}
            className="max-h-[70vh] overflow-y-auto px-3 sm:px-8 lg:px-10 py-5 lg:py-10 space-y-6 sm:space-y-8 scrollbar-hide bg-neutral-50/30 dark:bg-neutral-800/50"
          >
            <ChatMessageList
              currentStage={null}
              expandedProcessIds={expandedProcessIds}
              featuredAgentStep={null}
              isAssistantProcessing={false}
              isTyping={false}
              messages={messages}
              onCopy={(text) => navigator.clipboard.writeText(text)}
              onToggleProcessDetail={toggleProcessDetail}
              processingDurationLabel=""
              questionAnchorByMessageId={questionAnchorByMessageId}
              userAvatarUrl={data.shared_by.avatar_url}
              username={data.shared_by.username}
            />
          </div>
        </div>

        {/* Footer attribution */}
        <div className="mt-6 sm:mt-8 text-center">
          <p className="text-xs text-neutral-400 dark:text-neutral-500">
            {(() => {
              const brand = (
                <a href="/" className="font-bold text-primary-600 dark:text-primary-400 hover:underline">
                  {i18n.resolvedLanguage?.startsWith('en') ? 'WenDao AI' : '问道 AI'}
                </a>
              );
              const parts = t('sharedConversation.generatedBy', { brand: '__BRAND__' }).split('__BRAND__');
              return (
                <>
                  {parts[0]}
                  {brand}
                  {parts[1]}
                </>
              );
            })()}
          </p>
        </div>
      </div>
    </Layout>
  );
};
