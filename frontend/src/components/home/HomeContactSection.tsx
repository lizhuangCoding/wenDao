import { ContactLinks } from '@/components/common';

export const HomeContactSection = () => {
  return (
    <section className="mt-24 border-t border-neutral-100 pt-20 dark:border-neutral-800">
      <div className="rounded-[2rem] border border-neutral-200 bg-neutral-50 px-6 py-8 shadow-sm dark:border-neutral-800 dark:bg-neutral-950/40 sm:px-8">
        <div className="max-w-3xl">
          <p className="text-[11px] font-black uppercase tracking-[0.3em] text-primary-600 dark:text-primary-400">
            About me
          </p>
          <h2 className="mt-3 text-2xl font-serif font-black text-neutral-900 dark:text-neutral-100 sm:text-3xl">
            关于我
          </h2>
          <p className="mt-4 max-w-2xl text-sm leading-7 text-neutral-600 dark:text-neutral-400 sm:text-base">
            如果你想交流文章内容、技术方案，或者只是想认识一下我，可以直接通过下面的方式联系到我。
          </p>
        </div>

        <div className="mt-8">
          <ContactLinks />
        </div>
      </div>
    </section>
  );
};

