import { motion } from 'framer-motion';
import { ArticleContent } from '@/components/article';
import { AgentMoodIndicator } from './AgentMoodIndicator';
import { AgentProcessPanel } from './AgentProcessPanel';
import { ArticleReferencesPanel } from './ArticleReferencesPanel';
import { emptyReferenceGroups, parseChatArticleReferences } from '@/utils/chatReferences';
import type { ChatMessage, ChatStage, ChatStep } from '@/types';

interface ChatMessageListProps {
  currentStage: ChatStage | null;
  expandedProcessIds: Set<string>;
  featuredAgentStep: ChatStep | null;
  isAssistantProcessing: boolean;
  isTyping: boolean;
  messages: ChatMessage[];
  processingDurationLabel: string;
  questionAnchorByMessageId: Map<string, string>;
  userAvatarUrl?: string;
  username?: string;
  onCopy: (text: string) => void;
  onToggleProcessDetail: (id: string) => void;
}

export const ChatMessageList = ({
  currentStage,
  expandedProcessIds,
  featuredAgentStep,
  isAssistantProcessing,
  isTyping,
  messages,
  onCopy,
  onToggleProcessDetail,
  processingDurationLabel,
  questionAnchorByMessageId,
  userAvatarUrl,
  username,
}: ChatMessageListProps) => (
  <>
    {messages.map((message) => {
      const articleRefs = message.role === 'assistant'
        ? parseChatArticleReferences(message.content)
        : { body: message.content, references: emptyReferenceGroups() };
      const questionAnchorId = message.role === 'user'
        ? questionAnchorByMessageId.get(message.id)
        : undefined;

      return (
        <motion.div
          key={message.id}
          id={questionAnchorId}
          layout="position"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.22, ease: 'easeOut', layout: { duration: 0.2 } }}
          className={`scroll-mt-8 flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
        >
          <div className={`flex gap-4 ${message.role === 'user' ? 'max-w-[85%] flex-row-reverse' : 'w-full max-w-5xl flex-row'}`}>
            {message.role === 'user' ? (
              <div className="w-8 h-8 rounded-full overflow-hidden flex-shrink-0 border border-neutral-200 dark:border-neutral-600">
                <img
                  src={userAvatarUrl || `https://api.dicebear.com/7.x/avataaars/svg?seed=${username}`}
                  alt={username}
                  className="w-full h-full object-cover"
                />
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full bg-primary-500 flex-shrink-0 flex items-center justify-center">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
            )}

            <div className={`relative group/msg px-6 py-4 rounded-[24px] ${
              message.role === 'user'
                ? 'bg-neutral-900 dark:bg-neutral-700 text-white dark:text-neutral-100 rounded-tr-none shadow-elevated'
                : 'bg-white dark:bg-neutral-700 text-neutral-800 dark:text-neutral-100 border border-neutral-100 dark:border-neutral-600 rounded-tl-none shadow-sm'
            }`}>
              {message.role === 'assistant' ? (
                <div className="prose prose-sm max-w-none dark:prose-invert">
                  <AgentProcessPanel
                    messageId={message.id}
                    steps={message.processSteps || []}
                    expandedIds={expandedProcessIds}
                    onToggle={onToggleProcessDetail}
                  />
                  {message.content ? (
                    <>
                      {articleRefs.body && <ArticleContent content={articleRefs.body} />}
                      <ArticleReferencesPanel references={articleRefs.references} />
                    </>
                  ) : (
                    <p className="text-sm text-neutral-500 dark:text-neutral-400">正在生成最终回答...</p>
                  )}
                </div>
              ) : (
                <p className="text-sm leading-relaxed whitespace-pre-wrap font-medium">{message.content}</p>
              )}
              <div className="flex items-center justify-between mt-2">
                <span className={`text-[9px] font-bold uppercase tracking-tighter ${
                  message.role === 'user' ? 'text-neutral-400' : 'text-neutral-400 dark:text-neutral-500'
                }`}>
                  {new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
                <button
                  onClick={() => onCopy(message.content)}
                  className="opacity-0 group-hover/msg:opacity-100 p-1 text-neutral-400 hover:text-primary-500 transition-all ml-4"
                  title="复制内容"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </motion.div>
      );
    })}

    {isTyping && (
      <motion.div
        layout="position"
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: 'easeOut', layout: { duration: 0.2 } }}
        className="flex justify-start"
      >
        <div className="flex gap-4">
          <AgentMoodIndicator
            agentName={featuredAgentStep?.agent_name}
            detail={featuredAgentStep?.detail}
            showText={false}
            size="sm"
            stage={currentStage}
            status={featuredAgentStep?.status || 'running'}
            summary={featuredAgentStep?.summary}
          />
          <div className="bg-white dark:bg-neutral-700 border border-neutral-100 dark:border-neutral-600 px-6 py-4 rounded-[24px] rounded-tl-none shadow-sm">
            <AgentMoodIndicator
              agentName={featuredAgentStep?.agent_name}
              detail={featuredAgentStep?.detail}
              stage={currentStage}
              status={featuredAgentStep?.status || 'running'}
              summary={featuredAgentStep?.summary}
            />
            <p className="mt-3 text-[11px] font-bold text-neutral-500 dark:text-neutral-400">
              AI 助手处理中{isAssistantProcessing ? ` · 已耗时 ${processingDurationLabel}` : '...'}
            </p>
          </div>
        </div>
      </motion.div>
    )}
  </>
);
