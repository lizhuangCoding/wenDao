import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { siteApi } from '@/api';
import { Button, PageHeader, Panel, TextInput } from '@/components/common';
import { useUIStore } from '@/store';

export const Settings = () => {
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
      showToast('标语已更新', 'success');
      queryClient.setQueryData(['admin-slogan'], result);
      queryClient.invalidateQueries({ queryKey: ['slogan'] });
    },
    onError: (err: any) => {
      showToast(err.message || '保存失败', 'error');
    },
  });

  return (
    <div className="space-y-6">
      <PageHeader title="站点设置" description="管理网站基本配置，包括首页标语等。" />

      <Panel className="space-y-6">
        <div>
          <h3 className="text-lg font-bold text-neutral-800 dark:text-neutral-200 mb-1">首页标语</h3>
          <p className="text-sm text-neutral-500 dark:text-neutral-400 mb-4">
            首页展示的核心标语，修改后即刻生效。
          </p>
          <div className="grid gap-4 sm:grid-cols-[1fr_auto] items-end">
            <TextInput
              value={sloganInput}
              onChange={(e) => setSloganInput(e.target.value)}
              placeholder="请输入首页标语"
            />
            <Button
              onClick={() => {
                if (!sloganInput.trim()) {
                  showToast('标语不能为空', 'error');
                  return;
                }
                saveSloganMutation.mutate(sloganInput.trim());
              }}
              disabled={saveSloganMutation.isPending}
            >
              {saveSloganMutation.isPending ? '保存中...' : '保存标语'}
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  );
};
