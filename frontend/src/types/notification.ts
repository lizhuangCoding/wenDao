export type NotificationType =
  | 'comment_reply'
  | 'comment_like'
  | 'admin_broadcast'
  | 'system_notice';

export interface Notification {
  id: number;
  user_id: number;
  type: NotificationType;
  title: string;
  content: string;
  link_url: string;
  is_read: boolean;
  created_at: string;
}
