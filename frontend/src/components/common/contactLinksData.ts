import type { ContactLink } from '@/types';

export const defaultContactLinks: ContactLink[] = [
  {
    type: 'email',
    label: 'QQ 邮箱',
    value: '3174285493@qq.com',
    url: 'mailto:3174285493@qq.com',
    enabled: true,
    sort_order: 1,
  },
  {
    type: 'github',
    label: 'GitHub',
    value: 'lizhuangCoding',
    url: 'https://github.com/lizhuangCoding',
    enabled: true,
    sort_order: 2,
  },
];
