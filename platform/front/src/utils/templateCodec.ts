import pako from 'pako';

/** Keep legacy GZIP and plain JSON readable; write the existing UNIVER format. */
export function decodePrintTemplate(template: string): any {
  const prefix = template.startsWith('UNIVER:') ? 7 : template.startsWith('GZIP:') ? 5 : 0;
  if (!prefix) return JSON.parse(template);
  const bytes = Uint8Array.from(atob(template.slice(prefix)), char => char.charCodeAt(0));
  return JSON.parse(pako.inflate(bytes, { to: 'string' }));
}
export function encodePrintTemplate(workbook: unknown): string {
  const bytes = pako.gzip(JSON.stringify(workbook));
  let binary = '';
  // A large workbook must not exceed the argument limit of fromCharCode(...bytes).
  for (let offset = 0; offset < bytes.length; offset += 16384) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 16384));
  }
  return 'UNIVER:' + btoa(binary);
}
