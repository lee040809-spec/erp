export function focusInvalidField(form: HTMLFormElement) {
  const field = form.querySelector<HTMLElement>(
    ":invalid, [aria-invalid='true']",
  );
  let ancestor = field?.parentElement;
  while (ancestor && ancestor !== form) {
    if (ancestor instanceof HTMLDetailsElement) ancestor.open = true;
    ancestor = ancestor.parentElement;
  }
  field?.focus();
  field?.scrollIntoView?.({ block: "center", behavior: "smooth" });
}
