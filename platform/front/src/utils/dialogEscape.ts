// Reuse each dialog's existing cancel action, including its save/draft guards.
export function installDialogEscape(doc: Document) {
  const visible = (element: HTMLElement) =>
    !!element.getClientRects().length &&
    doc.defaultView?.getComputedStyle(element).visibility !== 'hidden';
  const layer = (element: HTMLElement) => {
    const levels: number[] = [];
    for (let parent: HTMLElement | null = element; parent; parent = parent.parentElement)
      if (doc.defaultView?.getComputedStyle(parent).zIndex !== 'auto')
        levels.unshift(Number(doc.defaultView?.getComputedStyle(parent).zIndex) || 0);
    return levels;
  };
  const keydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape' || event.isComposing || event.repeat) return;
    // Leave fullscreen explicitly; keep the preview and all underlying dialogs.
    if (doc.fullscreenElement) {
      event.preventDefault();
      event.stopImmediatePropagation();
      void doc.exitFullscreen().catch(() => {});
      return;
    }
    const dialogs = Array.from(doc.querySelectorAll<HTMLElement>('[data-escape-dialog]')).filter(visible);
    const top = dialogs.map((element, index) => ({ element, index, levels: layer(element) }))
      .sort((a, b) => {
        for (let i = 0; i < Math.max(a.levels.length, b.levels.length); i++) {
          const difference = (b.levels[i] || 0) - (a.levels[i] || 0);
          if (difference) return difference;
        }
        return b.index - a.index;
      })[0]?.element;
    if (!top) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    if (top.getAttribute('data-escape-blocked') === 'true') return;
    const cancel = Array.from(top.querySelectorAll<HTMLButtonElement>('button[data-escape-close]'))
      .find(button => button.closest('[data-escape-dialog]') === top && visible(button));
    // A busy or non-dismissible top layer must never close a dialog behind it.
    if (cancel && !cancel.matches(':disabled')) cancel.click();
  };
  doc.addEventListener('keydown', keydown, true);
  return () => doc.removeEventListener('keydown', keydown, true);
}
