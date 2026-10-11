import { Plus } from 'lucide-react';

export interface PrintField { key: string; label: string; isList?: boolean; }
const colors = {
  order: { button: 'border-zinc-200 hover:bg-zinc-50 hover:border-zinc-300', icon: 'text-zinc-400', code: 'text-zinc-400' },
  other: { button: 'border-purple-200 hover:bg-purple-50 hover:border-purple-300', icon: 'text-purple-400', code: 'text-purple-500' },
  item: { button: 'border-amber-200 hover:bg-amber-50 hover:border-amber-300', icon: 'text-amber-400', code: 'text-amber-500' },
};
export default function PrintTemplateFieldButtons({ fields, onSelect, tone }: {
  fields: PrintField[]; onSelect: (field: PrintField) => void; tone: keyof typeof colors;
}) {
  const style = colors[tone];
  return <>{fields.map(field => <button key={field.key} onClick={() => onSelect(field)}
    className={'inline-flex items-center gap-2 px-2 py-1.5 bg-white rounded-lg border transition-colors whitespace-nowrap w-[280px] ' + style.button}>
    <Plus className={'w-3 h-3 ' + style.icon} />
    <span className="text-xs text-zinc-700">{field.label}</span>
    <code className={'text-[10px] ml-2 ' + style.code}>{'{{' + (field.isList ? '#' : '') + field.key + '}}'}</code>
  </button>)}</>;
}
