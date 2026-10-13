import { useEffect, useRef, useState } from "react";
import {
  Download,
  ChevronLeft,
  ChevronRight,
  Maximize2,
  Minimize2,
  RotateCw,
  ZoomIn,
  ZoomOut,
  X,
} from "lucide-react";
import type { PDFDocumentProxy, RenderTask } from "pdfjs-dist";
import workerURL from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import { drawingBlob, PartDrawing } from "./api";
import "./drawings.css";
import { BundledPdfData } from "./pdfAssets";
export default function DrawingPreview({
  drawing,
  legacy,
  onClose,
}: {
  drawing?: PartDrawing;
  legacy?: string;
  onClose: () => void;
}) {
  const [blob, setBlob] = useState<Blob | null>(null),
    [url, setURL] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(true);
  const [doc, setDoc] = useState<PDFDocumentProxy | null>(null),
    [page, setPage] = useState(1),
    [zoom, setZoom] = useState(1),
    [rotation, setRotation] = useState(0),
    [frameWidth, setFrameWidth] = useState(800),
    [fullscreen, setFullscreen] = useState(false);
  const canvas = useRef<HTMLCanvasElement>(null),
    rendering = useRef<RenderTask | null>(null),
    frame = useRef<HTMLDivElement>(null),
    container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const target = frame.current;
    if (!target) return;
    const observer = new ResizeObserver(() => setFrameWidth(target.clientWidth));
    observer.observe(target);
    const changed = () => setFullscreen(document.fullscreenElement === container.current);
    document.addEventListener("fullscreenchange", changed);
    return () => {
      observer.disconnect();
      document.removeEventListener("fullscreenchange", changed);
    };
  }, []);
  useEffect(() => {
    const control = new AbortController();
    let objectURL = "";
    let loading: { destroy: () => Promise<void> } | undefined;
    let cancelled = false;
    setBusy(true);
    setError("");
    setDoc(null);
    setBlob(null);
    setURL("");
    setPage(1);
    setZoom(1);
    setRotation(0);
    (async () => {
      const file = legacy
        ? await (await fetch(legacy, { signal: control.signal })).blob()
        : await drawingBlob(drawing!.id, control.signal);
      if (cancelled) return;
      objectURL = URL.createObjectURL(file);
      setBlob(file);
      setURL(objectURL);
      if (
        drawing?.mime_type === "application/pdf" ||
        file.type === "application/pdf"
      ) {
        const pdfjs = await import("pdfjs-dist");
        pdfjs.GlobalWorkerOptions.workerSrc = workerURL;
        if (cancelled) return;
        const bytes = new Uint8Array(await file.arrayBuffer());
        if (cancelled) return;
        const task = pdfjs.getDocument({
          data: bytes,
          BinaryDataFactory: BundledPdfData,
          useWorkerFetch: false,
          cMapPacked: true,
          maxImageSize: 36000000,
          canvasMaxAreaInBytes: 64 * 1024 * 1024,
        });
        loading = task;
        const document = await task.promise;
        if (cancelled) {
          await task.destroy();
          return;
        }
        setDoc(document);
      }
      setBusy(false);
    })().catch((e) => {
      if (!cancelled) {
        setError(e instanceof Error ? e.message : "图纸读取失败");
        setBusy(false);
      }
    });
    return () => {
      cancelled = true;
      control.abort();
      rendering.current?.cancel();
      void loading?.destroy();
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [drawing?.id, legacy]);
  useEffect(() => {
    if (!doc || !canvas.current) return;
    let cancelled = false;
    let task: RenderTask | undefined;
    (async () => {
      const previous = rendering.current;
      previous?.cancel();
      try {
        await previous?.promise;
      } catch {}
      if (cancelled) return;
      const p = await doc.getPage(page);
      if (cancelled || !canvas.current) return;
      const pageRotation = (p.rotate + rotation) % 360;
      const base = p.getViewport({ scale: 1, rotation: pageRotation });
      const available = Math.max(240, frameWidth - 12);
      const output = Math.min(window.devicePixelRatio || 1, 2);
      const scale = Math.min(
        (available / base.width) * zoom,
        Math.sqrt(16000000 / (base.width * base.height * output * output)),
      );
      const viewport = p.getViewport({ scale, rotation: pageRotation });
      const target = canvas.current;
      target.width = Math.floor(viewport.width * output);
      target.height = Math.floor(viewport.height * output);
      target.style.width = `${viewport.width}px`;
      target.style.height = `${viewport.height}px`;
      task = p.render({
        canvas: target,
        canvasContext: target.getContext("2d")!,
        viewport,
        transform: output !== 1 ? [output, 0, 0, output, 0, 0] : undefined,
      });
      rendering.current = task;
      await task.promise;
    })().catch((e) => {
      if (!cancelled && e?.name !== "RenderingCancelledException")
        setError(e.message || "PDF页面读取失败");
    });
    return () => {
      cancelled = true;
      task?.cancel();
    };
  }, [doc, page, zoom, rotation, frameWidth]);
  const download = () => {
    if (!blob || !url) return;
    const link = document.createElement("a");
    link.href = url;
    link.download = drawing?.file_name || "零件图纸.png";
    link.click();
  };
  const isImage =
    drawing?.mime_type.startsWith("image/") || blob?.type.startsWith("image/");
  return (
    <div className="erp-overlay drawing-preview-overlay">
      <section
        ref={container}
        role="dialog"
        aria-modal="true"
        aria-label="图纸预览"
        className="drawing-preview"
      >
        <header>
          <div className="drawing-preview-title">
            <strong title={drawing?.file_name || "原有零件图纸"}>{drawing?.file_name || "原有零件图纸"}</strong>
            <small>
              {drawing?.drawing_number && `图号 ${drawing.drawing_number} · `}
              {drawing?.version && `版本 ${drawing.version} · `}
              {drawing && !drawing.is_current && drawing.order_item_id
                ? "历史版本 · 请核对"
                : ""}
            </small>
          </div>
          <button
            type="button"
            className="erp-btn"
            aria-label="关闭图纸预览"
            onClick={onClose}
          >
            <X size={18} />
          </button>
        </header>
        <div className="drawing-toolbar">
          {doc && (
            <>
              <button
                type="button"
                className="erp-btn"
                aria-label="上一页"
                title="上一页"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
              >
                <ChevronLeft size={16} />
              </button>
              <span>
                第 {page} / {doc.numPages} 页
              </span>
              <button
                type="button"
                className="erp-btn"
                aria-label="下一页"
                title="下一页"
                disabled={page >= doc.numPages}
                onClick={() => setPage(page + 1)}
              >
                <ChevronRight size={16} />
              </button>
            </>
          )}
          {(doc || isImage) && (
            <>
              <button
                type="button"
                className="erp-btn"
                aria-label="缩小图纸"
                disabled={zoom <= 0.5}
                onClick={() => setZoom(Math.max(0.5, zoom - 0.25))}
              >
                <ZoomOut size={16} />
              </button>
              <span>{Math.round(zoom * 100)}%</span>
              <button
                type="button"
                className="erp-btn"
                aria-label="放大图纸"
                disabled={zoom >= 3}
                onClick={() => setZoom(Math.min(3, zoom + 0.25))}
              >
                <ZoomIn size={16} />
              </button>
              <button
                type="button"
                className="erp-btn"
                aria-label="旋转图纸"
                onClick={() => setRotation((rotation + 90) % 360)}
              >
                <RotateCw size={16} />
              </button>
              <button
                type="button"
                className="erp-btn"
                onClick={() => {
                  setZoom(1);
                  setRotation(0);
                }}
              >
                适应宽度
              </button>
            </>
          )}
          <button
            type="button"
            className="erp-btn"
            disabled={!blob}
            onClick={download}
          >
            <Download size={16} />
            下载原文件
          </button>
          <button
            type="button"
            className="erp-btn"
            aria-label={fullscreen ? "退出全屏图纸" : "全屏图纸"}
            title={fullscreen ? "退出全屏" : "浏览器全屏"}
            onClick={() => {
              if (document.fullscreenElement) void document.exitFullscreen();
              else
                void container.current?.requestFullscreen?.().catch(() => {});
            }}
          >
            {fullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
          </button>
        </div>
        <div className="drawing-viewport" ref={frame}>
          {busy && <p role="status">正在读取图纸…</p>}
          {error && (
            <p role="alert" className="text-rose-600">
              {error}
            </p>
          )}
          {doc && <canvas aria-label={`PDF图纸第${page}页`} ref={canvas} />}{" "}
          {!busy && !doc && isImage && url && (
            <img
              src={url}
              alt={drawing?.file_name || "零件图纸"}
              style={{
                width: `${zoom * 100}%`,
                maxWidth: "none",
                transform: `rotate(${rotation}deg)`,
              }}
            />
          )}
          {!busy && !error && !doc && !isImage && blob && (
            <p>CAD原文件已保存。请下载查看，或另上传PDF用于在线预览。</p>
          )}
        </div>
      </section>
    </div>
  );
}
