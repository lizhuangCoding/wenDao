import { Suspense, lazy, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { DateRangePicker } from 'tdesign-react';
import { statApi } from '@/api';
import { Loading } from '@/components/common';
import { motion } from 'framer-motion';
import dayjs from 'dayjs';
import 'tdesign-react/es/style/index.css';

const DashboardChart = lazy(() =>
  import('./DashboardChart').then((module) => ({ default: module.DashboardChart }))
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
  const [queryType, setQueryType] = useState<QueryType>('7days');
  const [dateRange, setDateRange] = useState<[string, string]>(['', '']);
  const [startDateInput, setStartDateInput] = useState('');
  const [endDateInput, setEndDateInput] = useState('');

  const { data: stats, isLoading } = useQuery({
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

  // 准备图表数据
  const chartData: ChartDataPoint[] = stats?.daily_stat?.labels?.map((label: string, index: number) => ({
    fullDate: label,
    date: dayjs(label).format('M.DD'),
    weekday: dayjs(label).format('ddd'),
    pv: stats.daily_stat.pv[index] || 0,
    uv: stats.daily_stat.uv[index] || 0,
  })) || [];

  const peakPV = chartData.reduce((max: number, item: ChartDataPoint) => Math.max(max, item.pv), 0);
  const avgPV = chartData.length
    ? Math.round(chartData.reduce((sum: number, item: ChartDataPoint) => sum + item.pv, 0) / chartData.length)
    : 0;
  const activeRangeLabel = queryType === '7days'
    ? '近 7 天'
    : queryType === '30days'
      ? '近 30 天'
      : dateRange[0] && dateRange[1]
        ? `${dateRange[0]} 至 ${dateRange[1]}`
        : '自定义时间';

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
        alert('开始日期不能晚于结束日期');
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

  return (
    <div className="space-y-8">
      {/* 标题和筛选 */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4"
      >
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">
          {t('admin.dashboard')}
        </h1>
        <div className="w-full sm:w-auto rounded-2xl border border-neutral-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-3 shadow-sm">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="min-w-0 lg:pr-2">
              <div className="text-xs font-semibold text-neutral-700 dark:text-neutral-200">时间范围</div>
              <div className="text-[11px] text-neutral-400 dark:text-neutral-500">当前查看：{activeRangeLabel}</div>
            </div>

            <div className="grid grid-cols-3 gap-1 rounded-xl bg-neutral-100 dark:bg-neutral-800 p-1">
              <button
                type="button"
                onClick={() => handleQuickSelect('7days')}
                className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  queryType === '7days'
                    ? 'bg-white dark:bg-neutral-700 text-primary-600 dark:text-primary-300 shadow-sm'
                    : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100'
                }`}
              >
                {t('admin.recent7Days')}
              </button>
              <button
                type="button"
                onClick={() => handleQuickSelect('30days')}
                className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  queryType === '30days'
                    ? 'bg-white dark:bg-neutral-700 text-primary-600 dark:text-primary-300 shadow-sm'
                    : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100'
                }`}
              >
                {t('admin.recent30Days')}
              </button>
              <button
                type="button"
                onClick={handleCustomSelect}
                className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
                  queryType === 'custom'
                    ? 'bg-white dark:bg-neutral-700 text-primary-600 dark:text-primary-300 shadow-sm'
                    : 'text-neutral-500 dark:text-neutral-400 hover:text-neutral-800 dark:hover:text-neutral-100'
                }`}
              >
                自定义
              </button>
            </div>

            {queryType === 'custom' && (
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <div className="w-full sm:w-[320px] rounded-xl border border-neutral-200 dark:border-neutral-700 bg-neutral-50 dark:bg-neutral-800/60 p-1">
                  <DateRangePicker
                    value={startDateInput && endDateInput ? [startDateInput, endDateInput] : []}
                    valueType="YYYY-MM-DD"
                    format="YYYY-MM-DD"
                    placeholder={['开始日期', '结束日期']}
                    separator="至"
                    clearable
                    size="medium"
                    borderless
                    presets={{
                      '最近 7 天': [dayjs().subtract(6, 'day').format('YYYY-MM-DD'), dayjs().format('YYYY-MM-DD')],
                      '最近 30 天': [dayjs().subtract(29, 'day').format('YYYY-MM-DD'), dayjs().format('YYYY-MM-DD')],
                      '本月': [dayjs().startOf('month').format('YYYY-MM-DD'), dayjs().format('YYYY-MM-DD')],
                    }}
                    presetsPlacement="bottom"
                    popupProps={{ overlayClassName: 'wendao-date-range-popup' }}
                    onChange={(value) => {
                      const [start, end] = value;
                      setStartDateInput(start ? String(start) : '');
                      setEndDateInput(end ? String(end) : '');
                    }}
                    onClear={() => {
                      setStartDateInput('');
                      setEndDateInput('');
                    }}
                    style={{ width: '100%' }}
                  />
                </div>
                <button
                  type="button"
                  onClick={handleSearch}
                  disabled={!startDateInput || !endDateInput}
                  className="rounded-xl bg-primary-500 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:bg-neutral-300 dark:disabled:bg-neutral-700"
                >
                  {t('admin.query')}
                </button>
              </div>
            )}
          </div>
        </div>
      </motion.div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 p-8"
        >
          <div className="text-center">
            <div className="text-4xl font-bold text-primary-500 mb-2">{stats?.total_pv || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalPv')}</div>
          </div>
        </motion.div>
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 p-8"
        >
          <div className="text-center">
            <div className="text-4xl font-bold text-green-500 mb-2">{stats?.total_uv || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalUv')}</div>
          </div>
        </motion.div>
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 p-8"
        >
          <div className="text-center">
            <div className="text-4xl font-bold text-blue-500 mb-2">{stats?.total_comments || 0}</div>
            <div className="text-neutral-500 dark:text-neutral-400 text-sm font-medium">{t('admin.totalComments')}</div>
          </div>
        </motion.div>
      </div>

      {/* 流量趋势图 */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 p-8"
      >
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between mb-6">
          <div>
            <div className="text-xl font-semibold text-neutral-700 dark:text-neutral-200">{t('admin.trafficTrend')}</div>
            <div className="text-xs text-neutral-400 dark:text-neutral-500 mt-1">
              无访问的日期会以 0 补齐，方便观察连续趋势
            </div>
          </div>
          <div className="flex flex-wrap gap-2 text-xs">
            <span className="rounded-full bg-blue-50 dark:bg-blue-900/20 px-3 py-1 text-blue-600 dark:text-blue-300">
              峰值 PV {peakPV}
            </span>
            <span className="rounded-full bg-neutral-100 dark:bg-neutral-800 px-3 py-1 text-neutral-500 dark:text-neutral-300">
              日均 PV {avgPV}
            </span>
          </div>
        </div>
        <Suspense fallback={<div className="h-[400px] animate-pulse rounded-2xl bg-neutral-100 dark:bg-neutral-800" />}>
          <DashboardChart chartData={chartData} t={t} />
        </Suspense>
      </motion.div>
    </div>
  );
};
