import { request } from './client';
import type { ContactLink } from '@/types';

// 网站配置 API
export const siteApi = {
  // 获取网站标语
  getSlogan: (): Promise<{ slogan: string }> => {
    return request.get('/slogan');
  },

  // 获取联系方式
  getContactLinks: (): Promise<{ contact_links: ContactLink[] }> => {
    return request.get('/contact-links');
  },

  // 获取全站排序模式
  getSortMode: (): Promise<{ enabled: boolean }> => {
    return request.get('/settings/sort-mode');
  },

  // 设置全站排序模式（管理员）
  setSortMode: (enabled: boolean): Promise<void> => {
    return request.put('/admin/settings/sort-mode', { enabled });
  },

  // 设置网站标语（管理员）
  setSlogan: (slogan: string): Promise<{ slogan: string }> => {
    return request.put('/admin/settings/slogan', { slogan });
  },

  // 设置联系方式（管理员）
  setContactLinks: (contact_links: ContactLink[]): Promise<{ contact_links: ContactLink[] }> => {
    return request.put('/admin/settings/contact-links', { contact_links });
  },
};
