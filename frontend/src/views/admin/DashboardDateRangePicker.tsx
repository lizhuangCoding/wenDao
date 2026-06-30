import { DateRangePicker } from 'tdesign-react';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import 'tdesign-react/es/style/index.css';

interface DashboardDateRangePickerProps {
  endDateInput: string;
  startDateInput: string;
  onChange: (startDate: string, endDate: string) => void;
  onClear: () => void;
}

export const DashboardDateRangePicker = ({
  endDateInput,
  startDateInput,
  onChange,
  onClear,
}: DashboardDateRangePickerProps) => {
  const { t } = useTranslation();

  return (
    <div className="w-full rounded-xl border border-neutral-200 bg-neutral-50 p-1 dark:border-neutral-700 dark:bg-neutral-800/60 sm:w-[320px]">
      <DateRangePicker
        value={startDateInput && endDateInput ? [startDateInput, endDateInput] : []}
        valueType="YYYY-MM-DD"
        format="YYYY-MM-DD"
        placeholder={[t('admin.startDate'), t('admin.endDate')]}
        separator={t('common.to') || '至'}
        clearable
        size="medium"
        borderless
        presets={{
          [t('admin.recent7Days')]: [
            dayjs().subtract(6, 'day').format('YYYY-MM-DD'),
            dayjs().format('YYYY-MM-DD'),
          ],
          [t('admin.recent30Days')]: [
            dayjs().subtract(29, 'day').format('YYYY-MM-DD'),
            dayjs().format('YYYY-MM-DD'),
          ],
          [t('admin.thisMonth')]: [
            dayjs().startOf('month').format('YYYY-MM-DD'),
            dayjs().format('YYYY-MM-DD'),
          ],
        }}
        presetsPlacement="bottom"
        popupProps={{ overlayClassName: 'wendao-date-range-popup' }}
        onChange={(value) => {
          const [start, end] = value;
          onChange(start ? String(start) : '', end ? String(end) : '');
        }}
        onClear={onClear}
        style={{ width: '100%' }}
      />
    </div>
  );
};
