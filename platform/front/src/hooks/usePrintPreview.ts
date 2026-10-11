import { useEffect, useMemo, useRef } from 'react';
import { decodePrintTemplate } from '../utils/templateCodec';

export function usePrintWorkbook(template?: string) {
  return useMemo(() => {
    if (!template) return { workbook: null, error: '' };
    try { return { workbook: decodePrintTemplate(template), error: '' }; }
    catch { return { workbook: null, error: '模板内容无法解析，请重新选择或修复模板。' }; }
  }, [template]);
}

interface PreviewEditor { isReady?: () => boolean; }
export function usePrintPreview(mode: string, orderData: unknown, workbook: unknown,
  editor: { current: PreviewEditor | null }, fill: () => void) {
  const fillRef = useRef(fill);
  fillRef.current = fill;
  useEffect(() => {
    if (mode !== 'preview' || !orderData || !workbook) return;
    let stopped = false, timer: ReturnType<typeof setTimeout> | undefined;
    const checkReady = () => {
      if (stopped) return;
      if (editor.current?.isReady?.()) fillRef.current();
      else timer = setTimeout(checkReady, 100);
    };
    checkReady();
    // Changing templates or closing the preview must release the pending timer.
    return () => { stopped = true; if (timer !== undefined) clearTimeout(timer); };
  }, [mode, orderData, workbook, editor]);
}
