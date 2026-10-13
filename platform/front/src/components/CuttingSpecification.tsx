export const isCuttingProcess = (name: string) => name.trim().startsWith('下料');
export default function CuttingSpecification({ value, onChange, disabled = false, label = '下料尺寸规格' }: {
  value?: string | null; onChange: (value: string) => void; disabled?: boolean; label?: string;
}) {
  return <label className="erp-label w-full sm:col-span-3">下料尺寸规格（选填）
    <input type="text" className="erp-input" aria-label={label} maxLength={500} disabled={disabled}
      value={value ?? ''} placeholder="例如：Φ30 × 80 mm；板料 120 × 60 × 8 mm"
      onChange={e => onChange(e.target.value)} />
  </label>;
}
