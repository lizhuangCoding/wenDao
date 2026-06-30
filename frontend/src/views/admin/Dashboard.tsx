import { Suspense, lazy, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { statApi } from '@/api';
import { Button, ErrorState, Loading, PageHeader, Panel, SegmentedControl } from '@/components/common';
import { useUIStore } from '@/store';
import { getApiErrorMessage } from '@/utils/apiError';
import dayjs from 'dayjs';

const DashboardChart = lazy(() =>
  import('./DashboardChart').then((module) => ({ default: module.DashboardChart }))
);

const DashboardDateRangePicker = lazy(() =>
  import('./DashboardDateRangePicker').then((module) => ({
    default: module.DashboardDateRangePicker,
  }))
);

type QueryType = '7days' | '30days' | 'custom';

type ChartDataPoint = {
  fullDate: string;
  date: string;
  weekday: string;
  pv: number;
  uv: number;
};

export const Dashboard = () => {
  const { t } = useTranslation();
  const { showToast } = useUIStore();
  const [queryType, setQueryType] = useState<QueryType>('7days');
  const [dateRange, setDateRange] = useState<[string, string]>(['', '']);
  const [startDateInput, setStartDateInput] = useState('');
  const [endDateInput, setEndDateInput] = useState('');

  const {
    data: stats,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['dashboard-stats', queryType, dateRange],
    queryFn: () => {
      if (queryType === '7days') {
        return statApi.getDashboardStats(7);
      } else if (queryType === '30days') {
        return statApi.getDashboardStats(30);
      } else {
        const [start, end] = dateRange;
        return statApi.getDashboardStatsByRange(start, end);
      }
    },
    enabled: queryType !== 'custom' || (dateRange[0] !== '' && dateRange[1] !== ''),
  });

  const dailyStat = stats?.daily_stat;

  // 准备图表数据
  const chartData: ChartDataPoint[] = dailyStat?.labels?.map((label: string, index: number) => ({
    fullDate: label,
    date: dayjs(label).format('M.DD'),
    weekday: dayjs(label).format('ddd'),
    pv: dailyStat.pv[index] || 0,
    uv: dailyStat.uv[index] || 0,
  })) || [];

  const peakPV = chartData.reduce((max: number, item: ChartDataPoint) => Math.max(max, item.pv), 0);
  const avgPV = chartData.length
    ? Math.round(chartData.reduce((sum: number, item: ChartDataPoint) => sum + item.pv, 0) / chartData.length)
    : 0;
  const activeRangeLabel = queryType === '7days'
    ? t('admin.recent7Days')
    : queryType === '30days'
      ? t('admin.recent30Days')
      : dateRange[0] && dateRange[1]
        ? `${dateRange[0]} ${t('common.to')} ${dateRange[1]}`
        : t('admin.custom');

  const handleQuickSelect = (type: QueryType) => {
    setQueryType(type);
    setStartDateInput('');
    setEndDateInput('');
    if (type === '7days') {
      setDateRange(['', '']);
    } else if (type === '30days') {
      setDateRange(['', '']);
    }
  };

  const handleSearch = () => {
    if (startDateInput && endDateInput) {
      // 验证日期格式
      const start = dayjs(startDateInput).format('YYYY-MM-DD');
      const end = dayjs(endDateInput).format('YYYY-MM-DD');
      if (dayjs(start).isAfter(dayjs(end))) {
        showToast(t('dashboard.startAfterEnd'), 'error');
        return;
      }
      setDateRange([start, end]);
      setQueryType('custom');
    }
  };

  const handleCustomSelect = () => {
    setQueryType('custom');
    if (!startDateInput && !endDateInput) {
      setStartDateInput(dayjs().subtract(6, 'day').format('YYYY-MM-DD'));
      setEndDateInput(dayjs().format('YYYY-MM-DD'));
    }
  };

  if (isLoading) return <Loading />;
  if (isError) {
    return (
      <ErrorState
        title={t('dashboard.loadFailed')}
        message={getApiErrorMessage(error, t('dashboard.loadFailedDescription'))}
        onRetry={() => refetch()}
      />
    );
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title={t('admin.dashboard')}
        tone="admin"
        actions={
          <Panel padding="sm" variant="muted" className="w-full border-neutral-200 dark:border-neutral-700 sm:w-auto">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="min-w-0 lg:pr-2">
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">{t('admin.timeRange')}</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">{t('admin.currentViewing', { range: activeRangeLabel })}</div>
            </div>

            <SegmentedControl<QueryType>
              value={queryType}
              items={[
                { label: t('admin.recent7Days'), value: '7days' },
                { label: t('admin.recent30Days'), value: '30days' },
                { label: t('admin.custom'), value: 'custom' },
              ]}
              onChange={(value) => (value === 'custom' ? handleCustomSelect() : handleQuickSelect(value))}
            />

            {queryType === 'custom' && (
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <Suspense
                  fallback={
                    <div className="h-[46px] w-full animate-pulse rounded-xl bg-neutral-100 dark:bg-neutral-800 sm:w-[320px]" />
                  }
                >
                  <DashboardDateRangePicker
                    endDateInput={endDateInput}
                    startDateInput={startDateInput}
                    onChange={(start, end) => {
                      setStartDateInput(start);
                      setEndDateInput(end);
                    }}
                    onClear={() => {
                      setStartDateInput('');
                      setEndDateInput('');
                    }}
                  />
                </Suspense>
                <Button onClick={handleSearch} disabled={!startDateInput || !endDateInput}>
                  {t('admin.query')}
                </Button>
              </div>
            )}
          </div>
          </Panel>
        }
      />

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Panel padding="lg">
          <div className="text-center">
            <div className="text-4xl font-bold text-primary-500 mb-2">{stats?.total_pv || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalPv')}</div>
          </div>
        </Panel>
        <Panel padding="lg">
          <div className="text-center">
            <div className="text-4xl font-bold text-green-500 mb-2">{stats?.total_uv || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalUv')}</div>
          </div>
        </Panel>
        <Panel padding="lg">
          <div className="text-center">
            <div className="text-4xl font-bold text-blue-500 mb-2">{stats?.total_comments || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalComments')}</div>
          </div>
        </Panel>
      </div>

      {/* 流量趋势图 */}
      <Panel padding="lg">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between mb-6">
          <div>
            <div className="text-xl font-semibold text-neutral-700 dark:text-neutral-200">{t('admin.trafficTrend')}</div>
            <div className="text-xs text-neutral-400 dark:text-neutral-500 mt-1">
              {t('admin.noTrafficFill')}
            </div>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <span className="rounded-full bg-blue-50 dark:bg-blue-900/20 px-3 py-1 text-blue-600 dark:text-blue-300">
              {t('admin.peakPv', { count: peakPV })}
            </span>
            <span className="rounded-full bg-neutral-100 dark:bg-neutral-800 px-3 py-1 text-neutral-500 dark:text-neutral-300">
              {t('admin.avgPv', { count: avgPV })}
            </span>
          </div>
        </div>
        <Suspense fallback={<div className="h-[400px] animate-pulse rounded-2xl bg-neutral-100 dark:bg-neutral-800" />}>
          <DashboardChart chartData={chartData} t={t} />
        </Suspense>
      </Panel>
    </div>
  );
};
