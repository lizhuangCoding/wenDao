import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { aiObservabilityApi } from '@/api';
import {
  Button,
  BulkActionBar,
  ConfirmModal,
  DataTable,
  DataTableBody,
  DataTableCell,
  DataTableHeaderCell,
  DataTableHeadRow,
  DataTableRow,
  EmptyState,
  ErrorState,
  Loading,
  PageHeader,
  Pagination,
  Panel,
  SelectInput,
  StatusBadge,
  TextInput,
} from '@/components/common';
import { useUIStore } from '@/store';
import type { AIObservabilityRun } from '@/types';

type RunStatusFilter = '' | 'running' | 'completed' | 'failed' | 'waiting_user';

const formatDuration = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds <= 0) return '-';
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return rest ? `${minutes}m ${rest}s` : `${minutes}m`;
};

const formatCost = (run: AIObservabilityRun) => {
  const totalTokens = run.cost.prompt_tokens + run.cost.completion_tokens;
  if (run.cost.status === 'not_collected') return '未采集';
  if (run.cost.status === 'tokens_only') return `${totalTokens} tokens`;
  return `${totalTokens} tokens · ${run.cost.estimated_cost.toFixed(4)} ${run.cost.currency}`;
};

const statusVariant = (status: string): 'success' | 'warning' | 'danger' | 'neutral' => {
  if (status === 'completed') return 'success';
  if (status === 'running' || status === 'waiting_user') return 'warning';
  if (status === 'failed') return 'danger';
  return 'neutral';
};

const toolTotal = (run: AIObservabilityRun) =>
  run.tool_usage.local_search +
  run.tool_usage.web_search +
  run.tool_usage.web_fetch +
  run.tool_usage.doc_writer +
  run.tool_usage.other;

export const AIObservability = () => {
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(15);
  const [status, setStatus] = useState<RunStatusFilter>('');
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [expandedRunId, setExpandedRunId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [confirmBatchDelete, setConfirmBatchDelete] = useState(false);

  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['admin-ai-observability-runs', page, pageSize, status, keyword],
    queryFn: () =>
      aiObservabilityApi.listRuns({
        page,
        pageSize,
        status: status || undefined,
        keyword: keyword || undefined,
      }),
  });

  const runs = useMemo(() => data?.data ?? [], [data?.data]);
  const totalPages = Math.max(1, data?.totalPages ?? 1);
  const currentPageIds = runs.map((run) => run.id);
  const allCurrentPageSelected =
    currentPageIds.length > 0 && currentPageIds.every((id) => selectedIds.includes(id));
  const summary = useMemo(() => {
    return runs.reduce(
      (acc, run) => {
        acc.total += 1;
        if (run.status === 'failed' || run.failed_step_count > 0) acc.failed += 1;
        acc.steps += run.step_count;
        acc.tools += toolTotal(run);
        return acc;
      },
      { total: 0, failed: 0, steps: 0, tools: 0 }
    );
  }, [runs]);

  const batchDeleteMutation = useMutation({
    mutationFn: (ids: number[]) => aiObservabilityApi.batchDeleteRuns(ids),
    onSuccess: (result) => {
      showToast(`已删除 ${result.deleted_count} 条 AI 运行记录`, 'success');
      setSelectedIds([]);
      setConfirmBatchDelete(false);
      setExpandedRunId(null);
      queryClient.invalidateQueries({ queryKey: ['admin-ai-observability-runs'] });
      if (page > 1 && selectedIds.length >= runs.length) {
        setPage((currentPage) => Math.max(1, currentPage - 1));
      }
    },
    onError: (err: any) => showToast(err.message || '删除 AI 运行记录失败', 'error'),
  });

  const applySearch = (event: React.FormEvent) => {
    event.preventDefault();
    setKeyword(keywordInput.trim());
    setPage(1);
  };

  const resetFilters = () => {
    setStatus('');
    setKeyword('');
    setKeywordInput('');
    setPage(1);
    setExpandedRunId(null);
    setSelectedIds([]);
  };

  const toggleRunSelection = (id: number) => {
    setSelectedIds((ids) => (ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id]));
  };

  const toggleCurrentPageSelection = () => {
    setSelectedIds((ids) => {
      if (allCurrentPageSelected) {
        return ids.filter((id) => !currentPageIds.includes(id));
      }
      return Array.from(new Set([...ids, ...currentPageIds]));
    });
  };

  if (isLoading) return <Loading />;

  return (
    <div className="space-y-6">
      <PageHeader
        title="AI 观测"
        description="查看 ThinkTank 运行状态、工具调用、失败步骤和来源痕迹"
        tone="admin"
      />

      <div className="grid gap-4 md:grid-cols-4">
        <Panel padding="sm">
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">当前页运行</div>
          <div className="mt-2 text-3xl font-bold text-neutral-900 dark:text-neutral-100">{summary.total}</div>
        </Panel>
        <Panel padding="sm">
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">失败/异常</div>
          <div className="mt-2 text-3xl font-bold text-red-500">{summary.failed}</div>
        </Panel>
        <Panel padding="sm">
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">步骤数</div>
          <div className="mt-2 text-3xl font-bold text-primary-500">{summary.steps}</div>
        </Panel>
        <Panel padding="sm">
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">工具痕迹</div>
          <div className="mt-2 text-3xl font-bold text-emerald-500">{summary.tools}</div>
        </Panel>
      </div>

      <Panel className="space-y-3">
        <form onSubmit={applySearch} className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <TextInput
            value={keywordInput}
            onChange={(event) => setKeywordInput(event.target.value)}
            placeholder="搜索问题、归一化问题或错误信息"
            leading={<Search className="h-4 w-4" />}
          />
          <SelectInput
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as RunStatusFilter);
              setPage(1);
              setExpandedRunId(null);
            }}
          >
            <option value="">全部状态</option>
            <option value="running">运行中</option>
            <option value="waiting_user">等待用户</option>
            <option value="completed">已完成</option>
            <option value="failed">失败</option>
          </SelectInput>
          <div className="flex gap-2">
            <Button type="submit">搜索</Button>
            <Button variant="secondary" onClick={resetFilters}>重置</Button>
          </div>
        </form>
      </Panel>

      <BulkActionBar
        selectedCount={selectedIds.length}
        onDelete={() => setConfirmBatchDelete(true)}
        onClear={() => setSelectedIds([])}
        isDeleting={batchDeleteMutation.isPending}
        deleteLabel="删除运行记录"
      />

      {isError ? (
        <ErrorState message={(error as any)?.message || 'AI 运行记录加载失败'} onRetry={() => refetch()} />
      ) : (
        <>
          <DataTable
            minWidth="1360px"
            emptyState={
              runs.length === 0 ? (
                <EmptyState title="暂无 AI 运行记录" description="用户发起 AI 对话后，这里会显示运行过程。" className="m-6" />
              ) : null
            }
          >
            <thead>
              <DataTableHeadRow>
                <DataTableHeaderCell width="select">
                  <input
                    type="checkbox"
                    checked={allCurrentPageSelected}
                    onChange={toggleCurrentPageSelection}
                    className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                    aria-label="选择当前页 AI 运行记录"
                  />
                </DataTableHeaderCell>
                <DataTableHeaderCell width="compact">Run</DataTableHeaderCell>
                <DataTableHeaderCell width="wide">问题</DataTableHeaderCell>
                <DataTableHeaderCell width="compact">状态</DataTableHeaderCell>
                <DataTableHeaderCell width="compact">耗时</DataTableHeaderCell>
                <DataTableHeaderCell width="medium">工具</DataTableHeaderCell>
                <DataTableHeaderCell width="medium">Token</DataTableHeaderCell>
                <DataTableHeaderCell width="compact">来源</DataTableHeaderCell>
                <DataTableHeaderCell width="compact">失败</DataTableHeaderCell>
                <DataTableHeaderCell width="medium">创建时间</DataTableHeaderCell>
                <DataTableHeaderCell width="actionsCompact" align="right">操作</DataTableHeaderCell>
              </DataTableHeadRow>
            </thead>
            <DataTableBody>
              {runs.map((run) => (
                <DataTableRow key={run.id}>
                  <DataTableCell width="select" nowrap>
                    <input
                      type="checkbox"
                      checked={selectedIds.includes(run.id)}
                      onChange={() => toggleRunSelection(run.id)}
                      className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                      aria-label={`选择 AI 运行记录 ${run.id}`}
                    />
                  </DataTableCell>
                  <DataTableCell width="compact" nowrap>#{run.id}</DataTableCell>
                  <DataTableCell className="min-w-0">
                    <div className="line-clamp-2 text-sm text-neutral-700 dark:text-neutral-200" title={run.original_question}>
                      {run.original_question}
                    </div>
                    <div className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
                      会话 #{run.conversation_id} · 用户 #{run.user_id} · {run.current_stage || '-'}
                    </div>
                  </DataTableCell>
                  <DataTableCell width="compact" nowrap>
                    <StatusBadge variant={statusVariant(run.status)}>{run.status}</StatusBadge>
                  </DataTableCell>
                  <DataTableCell width="compact" nowrap>{formatDuration(run.duration_seconds)}</DataTableCell>
                  <DataTableCell width="medium">
                    <div className="flex flex-wrap gap-1 text-[11px] text-neutral-600 dark:text-neutral-300">
                      <span className="rounded bg-neutral-100 px-2 py-0.5 dark:bg-neutral-800">Local {run.tool_usage.local_search}</span>
                      <span className="rounded bg-neutral-100 px-2 py-0.5 dark:bg-neutral-800">Search {run.tool_usage.web_search}</span>
                      <span className="rounded bg-neutral-100 px-2 py-0.5 dark:bg-neutral-800">Fetch {run.tool_usage.web_fetch}</span>
                      <span className="rounded bg-neutral-100 px-2 py-0.5 dark:bg-neutral-800">Doc {run.tool_usage.doc_writer}</span>
                    </div>
                  </DataTableCell>
                  <DataTableCell width="medium">
                    <div className="text-xs text-neutral-600 dark:text-neutral-300">{formatCost(run)}</div>
                  </DataTableCell>
                  <DataTableCell width="compact" nowrap>
                    <span className="font-semibold text-emerald-600 dark:text-emerald-300">
                      {run.sources.quality_score}
                    </span>
                  </DataTableCell>
                  <DataTableCell width="compact" nowrap>
                    <span
                      className={run.failed_step_count > 0 || run.failure_category ? 'font-semibold text-red-500' : 'text-neutral-400'}
                      title={run.failure_category || undefined}
                    >
                      {run.failed_step_count}
                    </span>
                  </DataTableCell>
                  <DataTableCell width="medium" nowrap>{run.created_at}</DataTableCell>
                  <DataTableCell align="right" nowrap>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setExpandedRunId(expandedRunId === run.id ? null : run.id)}
                    >
                      {expandedRunId === run.id ? '收起' : '详情'}
                    </Button>
                  </DataTableCell>
                </DataTableRow>
              ))}
            </DataTableBody>
          </DataTable>

          {expandedRunId && (
            <RunDetail run={runs.find((run) => run.id === expandedRunId)} />
          )}

          <Pagination
            page={page}
            totalPages={totalPages}
            pageSize={pageSize}
            total={data?.total}
            onChange={(nextPage: number) => {
              setPage(nextPage);
              setExpandedRunId(null);
              setSelectedIds([]);
            }}
            onPageSizeChange={(nextPageSize) => {
              setPageSize(nextPageSize);
              setPage(1);
              setExpandedRunId(null);
              setSelectedIds([]);
            }}
          />

          <ConfirmModal
            isOpen={confirmBatchDelete}
            title="删除 AI 运行记录"
            message="确认删除选中的 AI 运行记录吗？这会同时删除对应的执行步骤日志，不会删除用户会话内容。"
            onConfirm={() => batchDeleteMutation.mutate(selectedIds)}
            onCancel={() => setConfirmBatchDelete(false)}
            isConfirming={batchDeleteMutation.isPending}
            isDanger
          />
        </>
      )}
    </div>
  );
};

const RunDetail = ({ run }: { run?: AIObservabilityRun }) => {
  if (!run) return null;

  return (
    <Panel className="space-y-5">
      <div className="grid gap-4 lg:grid-cols-3">
        <div>
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">来源摘要</div>
          <div className="mt-2 text-sm text-neutral-700 dark:text-neutral-200">
            站内命中 {run.sources.local_hits} · 外部痕迹 {run.sources.web_hits} · 质量 {run.sources.quality_score}
          </div>
          {run.sources.external_urls.length > 0 && (
            <div className="mt-2 space-y-1">
              {run.sources.external_urls.slice(0, 5).map((source) => (
                <a key={source.url} href={source.url} target="_blank" rel="noreferrer" className="block truncate text-xs text-primary-600 dark:text-primary-400">
                  {source.url} · {source.quality_score}
                </a>
              ))}
            </div>
          )}
        </div>
        <div>
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">Token / 成本</div>
          <div className="mt-2 text-sm text-neutral-700 dark:text-neutral-200">
            {formatCost(run)}
          </div>
          <div className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
            Prompt {run.cost.prompt_tokens} · Completion {run.cost.completion_tokens} · {run.cost.status}
          </div>
        </div>
        <div>
          <div className="text-xs font-semibold text-neutral-500 dark:text-neutral-400">用户反馈</div>
          <div className="mt-2 text-sm text-neutral-700 dark:text-neutral-200">
            {run.feedback.status === 'not_collected' ? '暂未采集反馈' : `评分 ${run.feedback.score}`}
          </div>
        </div>
      </div>

      {run.last_error && (
        <div className="rounded border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          {run.failure_category && (
            <div className="mb-1 text-xs font-bold uppercase tracking-wide">
              {run.failure_category} · {run.failure_fingerprint}
            </div>
          )}
          {run.last_error}
        </div>
      )}

      {run.failure_clusters.length > 0 && (
        <div className="flex flex-wrap gap-2 text-xs">
          {run.failure_clusters.map((cluster) => (
            <span key={cluster.category} className="rounded bg-red-50 px-2 py-1 font-semibold text-red-600 dark:bg-red-950/30 dark:text-red-300">
              {cluster.category} x{cluster.count}
            </span>
          ))}
        </div>
      )}

      {run.failed_steps.length > 0 && (
        <div>
          <div className="mb-2 text-sm font-semibold text-neutral-800 dark:text-neutral-100">失败步骤</div>
          <div className="space-y-2">
            {run.failed_steps.map((step) => (
              <div key={step.id} className="rounded border border-red-100 bg-red-50/60 p-3 dark:border-red-900/50 dark:bg-red-950/20">
                <div className="text-sm font-semibold text-red-700 dark:text-red-300">
                  {step.agent_name} · {step.type} · {step.category} · {step.created_at}
                </div>
                <div className="mt-1 text-sm text-neutral-700 dark:text-neutral-200">{step.summary}</div>
                {step.detail && <pre className="mt-2 max-h-36 overflow-auto whitespace-pre-wrap text-xs text-neutral-500 dark:text-neutral-400">{step.detail}</pre>}
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <div className="mb-2 text-sm font-semibold text-neutral-800 dark:text-neutral-100">过程步骤</div>
        <div className="grid gap-2">
          {run.steps.map((step) => (
            <div key={step.id} className="flex flex-col gap-1 rounded border border-neutral-200 p-3 dark:border-neutral-700 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <div className="text-sm font-medium text-neutral-800 dark:text-neutral-100">{step.summary}</div>
                <div className="text-xs text-neutral-400 dark:text-neutral-500">{step.agent_name} · {step.type} · {step.created_at}</div>
              </div>
              <StatusBadge variant={statusVariant(step.status)}>{step.status}</StatusBadge>
            </div>
          ))}
        </div>
      </div>
    </Panel>
  );
};
