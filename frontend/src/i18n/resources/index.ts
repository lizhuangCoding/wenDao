import { adminResources } from './admin';
import { articleResources } from './article';
import { authResources } from './auth';
import { chatResources } from './chat';
import { commonResources } from './common';

export const resources = {
  en: {
    translation: {
      ...commonResources.en,
      ...articleResources.en,
      ...authResources.en,
      ...chatResources.en,
      ...adminResources.en,
    },
  },
  zh: {
    translation: {
      ...commonResources.zh,
      ...articleResources.zh,
      ...authResources.zh,
      ...chatResources.zh,
      ...adminResources.zh,
    },
  },
} as const;
