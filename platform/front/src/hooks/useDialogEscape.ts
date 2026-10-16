import { useEffect } from 'react';
import { installDialogEscape } from '../utils/dialogEscape';

export function useDialogEscape() {
  useEffect(() => installDialogEscape(document), []);
}
