import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

type ChartDataPoint = {
  fullDate: string;
  date: string;
  weekday: string;
  pv: number;
  uv: number;
};

interface DashboardChartProps {
  chartData: ChartDataPoint[];
  t: (key: string) => string;
}

export const DashboardChart = ({ chartData, t }: DashboardChartProps) => (
  <div className="h-[400px]">
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={chartData} margin={{ top: 10, right: 12, left: 0, bottom: 18 }}>
        <defs>
          <linearGradient id="colorPV" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2} />
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="colorUV" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#22c55e" stopOpacity={0.2} />
            <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" className="dark:opacity-20" />
        <XAxis
          dataKey="date"
          interval={0}
          angle={-35}
          textAnchor="end"
          height={56}
          tick={{ fontSize: 12, fill: '#6b7280' }}
          axisLine={{ stroke: '#e5e7eb' }}
          tickLine={{ stroke: '#e5e7eb' }}
        />
        <YAxis
          tick={{ fontSize: 12, fill: '#6b7280' }}
          axisLine={{ stroke: '#e5e7eb' }}
          tickLine={{ stroke: '#e5e7eb' }}
        />
        <Tooltip
          labelFormatter={(_, payload) => payload?.[0]?.payload?.fullDate || ''}
          contentStyle={{
            backgroundColor: 'var(--tooltip-bg, #fff)',
            border: '1px solid var(--tooltip-border, #e5e7eb)',
            borderRadius: '8px',
            boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
          }}
          itemStyle={{ color: 'var(--tooltip-text, #374151)' }}
        />
        <Legend
          verticalAlign="bottom"
          height={36}
          iconType="circle"
          formatter={(value) => (
            <span className="text-neutral-600 dark:text-neutral-400 text-sm">
              {value === 'pv' ? t('admin.pvLabel') : t('admin.uvLabel')}
            </span>
          )}
        />
        <Area
          type="monotone"
          dataKey="pv"
          stroke="#3b82f6"
          strokeWidth={2}
          fillOpacity={1}
          fill="url(#colorPV)"
          name="pv"
        />
        <Area
          type="monotone"
          dataKey="uv"
          stroke="#22c55e"
          strokeWidth={2}
          fillOpacity={1}
          fill="url(#colorUV)"
          name="uv"
        />
      </AreaChart>
    </ResponsiveContainer>
  </div>
);
