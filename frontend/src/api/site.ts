import { request } from './client';

// 网站配置 API
export const siteApi = {
  // 获取网站标语
  getSlogan: (): Promise<{ slogan: string }> => {
    return request.get('/slogan');
  },

  // 获取全站排序模式
  getSortMode: (): Promise<{ enabled: boolean }> => {
    return request.get('/settings/sort-mode');
  },

  // 设置全站排序模式（管理员）
  setSortMode: (enabled: boolean): Promise<void> => {
    return request.put('/admin/settings/sort-mode', { enabled });
  },
};