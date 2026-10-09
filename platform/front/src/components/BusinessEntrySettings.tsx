import React, { useEffect, useState } from 'react';
import { authFetch } from './shared';
export interface EvidencePolicy {
    source_ref: boolean;
    review_notes: boolean;
    allocation_basis: boolean;
    closing_notes: boolean;
    operation_notes: boolean;
}
export const basicEvidence: EvidencePolicy = { source_ref: false, review_notes: false, allocation_basis: false, closing_notes: false, operation_notes: false };
export async function readEntryPolicy() {
    const res = await authFetch('/api/platform/business-entry-policy');
    const body = await res.json();
    if (!res.ok)
        throw new Error(body.message || body.error || '读取填写要求失败');
    return body.data as {
        effective: EvidencePolicy;
        can_configure: boolean;
        defaults: EvidencePolicy;
        role_modes: Record<string, string>;
        roles: {
            id: number;
            name: string;
        }[];
    };
}
export default function BusinessEntrySettings({ onSaved }: {
    onSaved?: () => void;
}) {
    const [data, setData] = useState<Awaited<ReturnType<typeof readEntryPolicy>> | null>(null), [open, setOpen] = useState(false), [error, setError] = useState(''), [saving, setSaving] = useState(false);
    const load = () => readEntryPolicy().then(setData).catch(e => setError(e.message));
    useEffect(() => { load(); }, []);
    if (!data?.can_configure)
        return null;
    const save = async (e: React.FormEvent) => {
        e.preventDefault();
        if (saving)
            return;
        setSaving(true);
        setError('');
        try {
            const res = await authFetch('/api/platform/business-entry-policy', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ defaults: data.defaults, role_modes: data.role_modes || {} }) });
            const body = await res.json();
            if (!res.ok)
                throw new Error(body.message || body.error || '保存失败');
            await load();
            setOpen(false);
            onSaved?.();
        }
        catch (e) {
            setError((e as Error).message);
        }
        finally {
            setSaving(false);
        }
    };
    return <><button type="button" className="erp-btn" onClick={() => { load(); setOpen(true); }}>填写要求设置</button>{open && <div className="erp-overlay z-[120]"><form onSubmit={save} role="dialog" aria-label="业务填写要求设置" className="erp-dialog p-6 space-y-5"><h3 className="text-xl font-bold">业务填写要求设置</h3><p className="text-sm text-slate-500">以下凭证信息默认选填。日期、数量、价格核对与异常情况仍须填写；这些设置不增加角色的功能权限。</p><h4 className="font-semibold">公司默认要求</h4>{(Object.entries({ source_ref: '原始批次依据编号', review_notes: '完工审核依据 / 备注', allocation_basis: '成本分摊依据', closing_notes: '月结审核依据', operation_notes: '报废与交货凭证 / 备注' }) as [
                keyof EvidencePolicy,
                string
            ][]).map(([key, label]) => <label key={key} className="flex items-center gap-3 text-sm"><input type="checkbox" checked={data.defaults[key] || false} onChange={e => setData({ ...data, defaults: { ...data.defaults, [key]: e.target.checked } })}/>{label}设为必填</label>)}<h4 className="font-semibold">按角色设置详细程度</h4>{(data.roles || []).map(role => <label key={role.id} className="erp-label flex items-center gap-3">{role.name}<select className="erp-input flex-1" value={data.role_modes?.[role.id] || 'inherit'} onChange={e => {
                    const modes = { ...data.role_modes };
                    if (e.target.value === 'inherit')
                        delete modes[role.id];
                    else
                        modes[role.id] = e.target.value;
                    setData({ ...data, role_modes: modes });
                }}><option value="inherit">遵循公司默认</option><option value="basic">简洁：凭证均选填</option><option value="detailed">详细：凭证均必填</option></select></label>)}{error && <p role="alert" className="text-rose-600">{error}</p>}<footer className="flex justify-end gap-2"><button type="button" disabled={saving} className="erp-btn" onClick={() => setOpen(false)}>取消</button><button disabled={saving} className="erp-btn erp-primary">{saving ? '保存中…' : '保存填写要求'}</button></footer></form></div>}</>;
}
