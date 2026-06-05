import { AgentMoodIndicator } from '@/components/chat/AgentMoodIndicator';
import type { ChatStep } from '@/types';

interface AgentProcessPanelProps {
  messageId: string;
  steps: ChatStep[];
  expandedIds: Set<string>;
  onToggle: (id: string) => void;
}

const statusText: Record<ChatStep['status'], string> = {
  running: '进行中',
  completed: '已完成',
  failed: '失败',
};

export const AgentProcessPanel = ({ messageId, steps, expandedIds, onToggle }: AgentProcessPanelProps) => {
  if (!steps.length) return null;

  return (
    <div className="mb-4 rounded-lg border border-neutral-200 dark:border-neutral-600 bg-neutral-50 dark:bg-neutral-800/70 overflow-hidden">
      <div className="px-4 py-3 border-b border-neutral-200 dark:border-neutral-600">
        <p className="text-xs font-bold text-neutral-700 dark:text-neutral-200">计划-执行-审查过程</p>
        <p className="text-[11px] text-neutral-500 dark:text-neutral-400 mt-1">
          默认展示摘要，展开可查看计划节点、工具调用、返回结果和原始日志。
        </p>
      </div>

      <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
        {steps.map((step) => {
          const key =
            step.id > 0
              ? `${messageId}-${step.id}`
              : `${messageId}-${step.run_id ?? 'runless'}-${step.agent_name}-${step.summary || step.type || 'step'}`;
          const isExpanded = expandedIds.has(key);
          const isRunning = step.status === 'running';
          const isFailed = step.status === 'failed';

          return (
            <div key={key} className="bg-white/70 dark:bg-neutral-800">
              <button
                type="button"
                onClick={() => onToggle(key)}
                className="w-full flex items-start gap-3 px-4 py-3 text-left hover:bg-neutral-50 dark:hover:bg-neutral-700/60 transition-colors"
                aria-expanded={isExpanded}
              >
                <AgentMoodIndicator
                  agentName={step.agent_name}
                  detail={step.detail}
                  showText={false}
                  size="sm"
                  status={isFailed ? 'failed' : isRunning ? 'running' : step.status}
                  summary={step.summary}
                />
                <span className="min-w-0 flex-1">
                  <span className="block text-xs font-bold text-neutral-800 dark:text-neutral-100">
                    {step.summary || `${step.agent_name} 正在协作`}
                  </span>
                  <span className="mt-1 block text-[11px] text-neutral-500 dark:text-neutral-400">
                    {step.agent_name} · {statusText[step.status] || step.status}
                  </span>
                </span>
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className={`h-4 w-4 flex-shrink-0 text-neutral-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>

              {isExpanded && (
                <div className="px-4 pb-4 pl-9">
                  <pre className="max-h-72 overflow-auto rounded-lg bg-neutral-950 text-neutral-100 p-3 text-[11px] leading-relaxed whitespace-pre-wrap">
                    {step.detail || '暂无详细过程日志。'}
                  </pre>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
