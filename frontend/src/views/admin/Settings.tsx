import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react';
import { siteApi } from '@/api';
import { Button, PageHeader, Panel, SelectInput, TextInput, ToggleSwitch } from '@/components/common';
import { defaultContactLinks } from '@/components/common/contactLinksData';
import { useUIStore } from '@/store';
import type { ContactLink } from '@/types';

const contactLinkTypeOptions = [
  { value: 'email', label: 'Email' },
  { value: 'github', label: 'GitHub' },
  { value: 'wechat', label: 'WeChat' },
  { value: 'link', label: 'Link' },
];

const createEmptyContactLink = (sortOrder: number): ContactLink => ({
  type: 'link',
  label: '',
  value: '',
  url: '',
  enabled: true,
  sort_order: sortOrder,
});

const reindexContactLinks = (links: ContactLink[]) =>
  links.map((link, index) => ({
    ...link,
    sort_order: index + 1,
  }));

const normalizeContactLink = (link: ContactLink): ContactLink => {
  const label = link.label.trim();
  const value = link.value.trim();
  const explicitUrl = link.url?.trim() || '';
  let url = explicitUrl;

  if (!url && value) {
    if (link.type === 'email') {
      url = `mailto:${value}`;
    } else if (link.type === 'github') {
      url = `https://github.com/${value.replace(/^@/, '')}`;
    }
  }

  return {
    ...link,
    type: link.type.trim() || 'link',
    label,
    value,
    url,
    enabled: Boolean(link.enabled),
  };
};

export const Settings = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [sloganInput, setSloganInput] = useState('');
  const [contactLinksInput, setContactLinksInput] = useState<ContactLink[]>(defaultContactLinks);

  const { data: sloganData } = useQuery({
    queryKey: ['admin-slogan'],
    queryFn: siteApi.getSlogan,
  });

  const { data: contactLinksData } = useQuery({
    queryKey: ['contact-links'],
    queryFn: siteApi.getContactLinks,
  });

  useEffect(() => {
    if (sloganData?.slogan) {
      setSloganInput(sloganData.slogan);
    }
  }, [sloganData]);

  useEffect(() => {
    const sourceLinks = contactLinksData?.contact_links ?? defaultContactLinks;
    setContactLinksInput(reindexContactLinks(sourceLinks.map((link) => ({ ...link }))));
  }, [contactLinksData]);

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

  const saveContactLinksMutation = useMutation({
    mutationFn: (contactLinks: ContactLink[]) => siteApi.setContactLinks(contactLinks),
    onSuccess: (result) => {
      showToast(t('settings.contactLinksUpdated'), 'success');
      queryClient.setQueryData(['contact-links'], result);
    },
    onError: (err: any) => {
      showToast(err.message || t('settings.saveFailed'), 'error');
    },
  });

  const normalizedContactLinks = useMemo(
    () => reindexContactLinks(contactLinksInput.map((link) => normalizeContactLink(link))),
    [contactLinksInput]
  );

  const updateContactLink = (index: number, patch: Partial<ContactLink>) => {
    setContactLinksInput((current) =>
      current.map((link, currentIndex) =>
        currentIndex === index
          ? { ...link, ...patch }
          : link
      )
    );
  };

  const addContactLink = () => {
    setContactLinksInput((current) => [...current, createEmptyContactLink(current.length + 1)]);
  };

  const removeContactLink = (index: number) => {
    setContactLinksInput((current) => reindexContactLinks(current.filter((_, currentIndex) => currentIndex !== index)));
  };

  const moveContactLink = (index: number, direction: -1 | 1) => {
    setContactLinksInput((current) => {
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= current.length) return current;

      const next = [...current];
      [next[index], next[nextIndex]] = [next[nextIndex], next[index]];
      return reindexContactLinks(next);
    });
  };

  const handleSaveContactLinks = () => {
    const hasInvalidRow = normalizedContactLinks.some(
      (link) => !link.label.trim() || !link.value.trim()
    );

    if (hasInvalidRow) {
      showToast(t('settings.contactLinksRequired'), 'error');
      return;
    }

    saveContactLinksMutation.mutate(normalizedContactLinks);
  };

  return (
    <div className="space-y-6">
      <PageHeader title={t('settings.title')} description={t('settings.description')} />

      <Panel className="space-y-6">
        <div>
          <h3 className="mb-1 text-lg font-bold text-neutral-800 dark:text-neutral-200">{t('settings.sloganTitle')}</h3>
          <p className="mb-4 text-sm text-neutral-500 dark:text-neutral-400">
            {t('settings.sloganHint')}
          </p>
          <div className="grid items-end gap-4 sm:grid-cols-[1fr_auto]">
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

      <Panel className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h3 className="mb-1 text-lg font-bold text-neutral-800 dark:text-neutral-200">{t('settings.contactLinksTitle')}</h3>
            <p className="text-sm text-neutral-500 dark:text-neutral-400">{t('settings.contactLinksHint')}</p>
          </div>
          <Button variant="secondary" onClick={addContactLink}>
            <Plus className="h-4 w-4" />
            {t('settings.addContactLink')}
          </Button>
        </div>

        <div className="space-y-4">
          {contactLinksInput.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-neutral-200 px-6 py-10 text-center text-sm text-neutral-400 dark:border-neutral-700 dark:text-neutral-500">
              {t('settings.contactLinksEmpty')}
            </div>
          ) : (
            contactLinksInput.map((link, index) => (
              <div
                key={`${link.type}-${link.sort_order}-${index}`}
                className="rounded-2xl border border-neutral-100 bg-neutral-50/70 p-4 dark:border-neutral-800 dark:bg-neutral-900/50"
              >
                <div className="flex items-start gap-4">
                  <div className="grid min-w-0 flex-1 gap-4 md:grid-cols-2">
                    <div>
                      <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                        {t('settings.contactLinkType')}
                      </label>
                      <SelectInput
                        value={link.type}
                        onChange={(e) => updateContactLink(index, { type: e.target.value })}
                      >
                        {contactLinkTypeOptions.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </SelectInput>
                    </div>

                    <div>
                      <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                        {t('settings.contactLinkLabel')}
                      </label>
                      <TextInput
                        value={link.label}
                        onChange={(e) => updateContactLink(index, { label: e.target.value })}
                        placeholder={t('settings.contactLinkLabelPlaceholder')}
                      />
                    </div>

                    <div>
                      <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                        {t('settings.contactLinkValue')}
                      </label>
                      <TextInput
                        value={link.value}
                        onChange={(e) => updateContactLink(index, { value: e.target.value })}
                        placeholder={t('settings.contactLinkValuePlaceholder')}
                      />
                    </div>

                    <div>
                      <label className="mb-2 block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                        {t('settings.contactLinkUrl')}
                      </label>
                      <TextInput
                        value={link.url || ''}
                        onChange={(e) => updateContactLink(index, { url: e.target.value })}
                        placeholder={t('settings.contactLinkUrlPlaceholder')}
                      />
                    </div>
                  </div>

                  <div className="flex shrink-0 items-center gap-1 pt-8">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => moveContactLink(index, -1)}
                      disabled={index === 0}
                      aria-label={t('settings.moveContactLinkUp')}
                    >
                      <ArrowUp className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => moveContactLink(index, 1)}
                      disabled={index === contactLinksInput.length - 1}
                      aria-label={t('settings.moveContactLinkDown')}
                    >
                      <ArrowDown className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => removeContactLink(index)}
                      aria-label={t('settings.removeContactLink')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <div className="mt-4 flex items-center justify-between border-t border-neutral-100 pt-4 dark:border-neutral-800">
                  <div className="text-xs font-semibold uppercase tracking-[0.24em] text-neutral-400 dark:text-neutral-500">
                    {t('settings.contactLinkOrder', { order: link.sort_order })}
                  </div>
                  <ToggleSwitch
                    checked={link.enabled}
                    onClick={() => updateContactLink(index, { enabled: !link.enabled })}
                  />
                </div>
              </div>
            ))
          )}
        </div>

        <div className="flex justify-end">
          <Button
            onClick={handleSaveContactLinks}
            disabled={saveContactLinksMutation.isPending}
          >
            {saveContactLinksMutation.isPending ? t('common.saving') : t('settings.saveContactLinks')}
          </Button>
        </div>
      </Panel>
    </div>
  );
};
