import type { Company } from '../types';
import { authFetch } from '../utils/auth';

/** Portal authentication is separate from the ERP business application's session. */
export async function fetchPortalCompanies(): Promise<Company[]> {
  const response = await authFetch('/api/portal/companies');
  if (!response.ok) throw new Error('公司列表加载失败');
  const data = await response.json();
  return Array.isArray(data) ? data : [];
}
