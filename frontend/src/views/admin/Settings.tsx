import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { siteApi } from '@/api';
import { Button, PageHeader, Panel, TextInput } from '@/components/common';
import { useUIStore } from '@/store';

export const Settings = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [sloganInput, setSloganInput] = useState('');

  const { data: sloganData } = useQuery({
    queryKey: ['admin-slogan'],
    queryFn: siteApi.getSlogan,
  });

  useEffect(() => {
    if (sloganData?.slogan) {
      setSloganInput(sloganData.slogan);
    }
  }, [sloganData]);

  const saveSloganMutation = useMutation({
    mutationFn: (slogan: string) => siteApi.setSlogan(slogan),
    onSuccess: (result) => {
      showToast(t('settings.sloganUpdated'), 'success');
      queryClient.setQueryData(['admin-slogan'], result);
      queryClient.invalidateQueries({ queryKey: ['slogan'] });
    },
    onError: (err: any) => {
      showToast(err.message || t('settings.saveFailed'), 'error');
    },
  });

  return (
    <div className="space-y-6">
      <PageHeader title={t('settings.title')} description={t('settings.description')} />

      <Panel className="space-y-6">
        <div>
          <h3 className="text-lg font-bold text-neutral-800 dark:text-neutral-200 mb-1">{t('settings.sloganTitle')}</h3>
          <p className="text-sm text-neutral-500 dark:text-neutral-400 mb-4">
            {t('settings.sloganHint')}
          </p>
          <div className="grid gap-4 sm:grid-cols-[1fr_auto] items-end">
            <TextInput
              value={sloganInput}
              onChange={(e) => setSloganInput(e.target.value)}
              placeholder={t('settings.sloganPlaceholder')}
            />
            <Button
              onClick={() => {
                if (!sloganInput.trim()) {
                  showToast(t('settings.sloganRequired'), 'error');
                  return;
                }
                saveSloganMutation.mutate(sloganInput.trim());
              }}
              disabled={saveSloganMutation.isPending}
            >
              {saveSloganMutation.isPending ? t('common.saving') : t('settings.saveSlogan')}
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  );
};
