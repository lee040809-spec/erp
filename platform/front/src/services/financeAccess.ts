import {authFetch} from '../components/shared';
export interface FinanceAccess {can_import_history:boolean;can_manage_costs:boolean}
export async function fetchFinanceAccess():Promise<FinanceAccess>{
 const res=await authFetch('/api/platform/finance/access');const body=await res.json();
 if(!res.ok)throw new Error(body.message||body.error||'读取产值操作权限失败');return body.data;
}
