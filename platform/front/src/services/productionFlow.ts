import { businessToday } from '../utils/businessDate';
import { authFetch } from '../components/shared';
export async function flowRequest<T>(url: string, method = 'GET', payload?: unknown): Promise<T> {
  const response = await authFetch(`/api/platform${url}`, { method, ...(payload === undefined ? {} : { headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) }) });
  const body = await response.json();
  if (!response.ok || (body.code !== undefined && body.code !== 0)) throw new Error(body.message || body.error || '操作失败');
  return body.data;
}
export const flowDate = (date?: string | null) => date?.slice(0, 10) || '';
export const flowToday = businessToday;
// A retry of the same payload keeps its request ID, preventing duplicate reports after network timeouts.
export function withRequestID(state: { signature: string; id: string }, payload: object) {
  const signature = JSON.stringify(payload);
  if (signature !== state.signature) { state.signature = signature; state.id = crypto.randomUUID(); }
  return { ...payload, request_id: state.id };
}
