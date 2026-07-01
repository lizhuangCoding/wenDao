export interface User {
  id: number;
  username: string;
  email: string;
  avatar_url?: string;
  bio?: string;
  role: 'user' | 'admin';
  status: 'active' | 'banned';
  comment_reply_email_enabled?: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  verification_code: string;
}

export interface VerificationCodeRequest {
  email: string;
}

export interface PasswordResetConfirmRequest {
  email: string;
  password: string;
  verification_code: string;
}

export interface UpdatePreferencesRequest {
  comment_reply_email_enabled: boolean;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface CurrentUserResponse {
  user: User;
  expires_in: number;
}
