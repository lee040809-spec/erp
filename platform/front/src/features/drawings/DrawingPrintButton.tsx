import { useEffect, useRef, useState } from "react";
import { Printer } from "lucide-react";
import type { PDFDocumentProxy } from "pdfjs-dist";
import { printDrawing } from "./printDrawing";
import "./drawings.css";

export default function DrawingPrintButton({
  pdf,
  imageUrl,
  title,
  rotation = 0,
  disabled = false,
  pageNumber,
}: {
  pdf?: PDFDocumentProxy | null;
  imageUrl?: string;
  title: string;
  rotation?: number;
  disabled?: boolean;
  pageNumber?: number;
}) {
  const [printing, setPrinting] = useState(false),
    [error, setError] = useState("");
  const request = useRef<AbortController | null>(null);
  useEffect(() => {
    setError("");
    setPrinting(false);
    return () => {
      request.current?.abort();
      request.current = null;
    };
  }, [pdf, imageUrl]);
  const [range, setRange] = useState("all");
  const print = async () => {
    if (disabled || request.current || (!pdf && !imageUrl)) return;
    const control = new AbortController();
    request.current = control;
    setError("");
    setPrinting(true);
    try {
      await printDrawing({
        pdf,
        imageUrl,
        title,
        rotation,
        pageNumber: range === "current" ? pageNumber : undefined,
        signal: control.signal,
      });
    } catch (error) {
      if (!control.signal.aborted)
        setError(error instanceof Error ? error.message : "打印失败，请重试");
    } finally {
      if (request.current === control) {
        request.current = null;
        setPrinting(false);
      }
    }
  };
  return (
    <>
      {pdf && pageNumber && (
        <select
          aria-label="图纸打印范围"
          className="erp-input !w-auto"
          disabled={printing}
          value={range}
          onChange={(e) => setRange(e.target.value)}
        >
          <option value="all">全部页</option>
          <option value="current">当前页（第{pageNumber}页）</option>
        </select>
      )}
      <button
        type="button"
        className="erp-btn"
        aria-label="打印图纸"
        title="打印当前所选版本；请在打印窗口核对纸张与比例。加工尺寸请以标注为准"
        disabled={disabled || printing || (!pdf && !imageUrl)}
        onClick={print}
      >
        <Printer size={16} />
        {printing ? "正在打印…" : "打印图纸"}
      </button>
      {error && (
        <span role="alert" className="drawing-print-error">
          {error}
        </span>
      )}
    </>
  );
}
