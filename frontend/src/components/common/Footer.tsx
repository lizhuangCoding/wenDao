import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { siteApi } from '@/api';
import { ContactLinks } from './ContactLinks';
import { defaultContactLinks } from './contactLinksData';

export const Footer = () => {
  const { t } = useTranslation();
  const { data: contactLinksData } = useQuery({
    queryKey: ['contact-links'],
    queryFn: siteApi.getContactLinks,
    staleTime: 5 * 60 * 1000,
  });

  return (
    <footer className="mt-20 border-t border-neutral-200 bg-white dark:border-primary-900/20 dark:bg-[#050a10]">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,420px)] lg:items-center">
          <div className="text-center text-sm text-neutral-500 dark:text-neutral-400 lg:text-left">
            <p className="font-serif italic">芝兰生于深林，不以无人而不芳，君子修道立德，不为穷困而改节。</p>
          </div>

          <div className="lg:justify-self-end">
            <div className="mb-3 text-center text-[11px] font-black uppercase tracking-[0.32em] text-neutral-400 dark:text-neutral-500 lg:text-left">
              {t('common.contactMe')}
            </div>
            <ContactLinks links={contactLinksData?.contact_links ?? defaultContactLinks} />
          </div>
        </div>
      </div>
    </footer>
  );
};
