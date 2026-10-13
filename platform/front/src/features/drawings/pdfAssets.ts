// Keep font maps and image decoders in the deployment bundle; never send private PDFs to an external viewer.
const files = import.meta.glob<string>(
  [
    "/node_modules/pdfjs-dist/cmaps/*.bcmap",
    "/node_modules/pdfjs-dist/standard_fonts/*.{ttf,pfb}",
    "/node_modules/pdfjs-dist/wasm/*.wasm",
  ],
  { eager: true, query: "?url", import: "default" },
);
const directories: Record<string, string> = {
  cMapUrl: "cmaps",
  standardFontDataUrl: "standard_fonts",
  wasmUrl: "wasm",
};
export class BundledPdfData {
  async fetch({ kind, filename }: { kind: string; filename: string }) {
    const directory = directories[kind];
    const url =
      directory && files[`/node_modules/pdfjs-dist/${directory}/${filename}`];
    if (!url) throw new Error("图纸解码资源不存在：" + filename);
    const response = await fetch(url);
    if (!response.ok) throw new Error("图纸解码资源读取失败");
    return new Uint8Array(await response.arrayBuffer());
  }
}
