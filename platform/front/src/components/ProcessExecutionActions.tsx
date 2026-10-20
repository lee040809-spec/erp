import { CheckCircle2, Play } from 'lucide-react';
import type { OrderProcess } from '../types';

export default function ProcessExecutionActions({ process, disabled, onStatus }: {
  process: OrderProcess;
  disabled: boolean;
  onStatus: (status: 'processing' | 'completed') => void;
}) {
  if (process.status === 'completed') return null;
  return <>
    {process.status === 'pending' && <button type="button" className="erp-btn" disabled={disabled} onClick={() => onStatus('processing')}><Play size={15}/>现在开工</button>}
    <button type="button" className="erp-btn erp-success" disabled={disabled} onClick={() => onStatus('completed')}><CheckCircle2 size={15}/>当前工序完工</button>
  </>;
}
