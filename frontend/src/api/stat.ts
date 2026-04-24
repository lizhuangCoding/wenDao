import { request } from './client';

// 统计 API
export const statApi = {
  // 获取后台统计面板数据（按天数）
  getDashboardStats: (days: number = 7) => {
    return request.get(`/admin/stats/dashboard`, { params: { days } });
  },

  // 获取后台统计面板数据（按日期范围）
  getDashboardStatsByRange: (startDate: string, endDate: string) => {
    return request.get(`/admin/stats/dashboard`, { params: { start_date: startDate, end_date: endDate } });
  },
};