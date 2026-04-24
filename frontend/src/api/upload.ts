import { request } from './client';
import type { User } from '@/types';

export interface UploadResponse {
  url: string;
  filename: string;
  size: number;
}

export const uploadApi = {
  // 上传图片（管理员）
  uploadImage: (file: File, usage: 'cover' | 'content' = 'content') => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('usage', usage);
    return request.post<UploadResponse>('/admin/upload/image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },

  uploadAvatar: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return request.post<User>('/users/me/avatar', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
};
