export interface Company {
  id: number;
  name: string;
  code: string;
  description: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: number;
  username: string;
  email: string;
  company_id: number;
  company_name: string;
  role_id: number;
  role_name: string;
  login_attempts: number;
  lock_until: string | null;
  created_at: string;
  updated_at: string;
}

export interface SuperUser {
  id: number;
  username: string;
  email: string;
  role_id: number;
  role_name: string;
  login_attempts: number;
  lock_until: string | null;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: number;
  name: string;
  company_id: number;
  company_name: string;
  description: string;
  created_at: string;
  resources?: Resource[];
}

export interface Resource {
  id: number;
  name: string;
  type: 'menu' | 'button' | 'api';
  path: string;
  parent_id: number;
  sort_order: number;
}

export interface AuthState {
  authenticated: boolean;
  user: User | null;
  token: string | null;
}