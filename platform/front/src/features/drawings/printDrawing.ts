import type { PDFDocumentProxy } from "pdfjs-dist";

export interface DrawingPrintOptions {
  pdf?: PDFDocumentProxy | null;
  imageUrl?: string;
  title: string;
  rotation?: number;
  pageNumber?: number;
  signal: AbortSignal;
}

// Render into a separate same-origin document so only the drawing is printed.
// Reuse the authenticated preview and bundled PDF decoder; no upload or new fetch.
export async function printDrawing({
  pdf,
  imageUrl,
  title,
  rotation = 0,
  pageNumber,
  signal,
}: DrawingPrintOptions) {
  signal.throwIfAborted();
  if (!pdf && !imageUrl) throw new Error("请先等待图纸读取完成");
  const frame = document.createElement("iframe");
  frame.title = "图纸打印";
  frame.setAttribute("aria-hidden", "true");
  frame.style.cssText =
    "position:fixed;left:-10000px;top:0;width:1px;height:1px;border:0;";
  (document.fullscreenElement || document.body).appendChild(frame);
  const urls: string[] = [];
  try {
    const target = frame.contentDocument,
      view = frame.contentWindow;
    if (!target || !view)
      throw new Error("无法打开打印窗口，请下载原文件后打印");
    target.open();
    target.write(
      '<!doctype html><html><head><meta charset="utf-8"></head><body></body></html>',
    );
    target.close();
    target.title = title;
    const style = target.createElement("style");
    style.textContent =
      "html,body{margin:0;padding:0;background:white}.sheet{display:flex;align-items:center;justify-content:center;overflow:hidden;break-inside:avoid;break-after:page}.sheet:last-child{break-after:auto}.sheet img{display:block;object-fit:contain;flex:none}";
    target.head.appendChild(style);
    const appendPage = (
      src: string,
      width: number,
      height: number,
      imageRotation = 0,
    ) => {
      const sheet = target.createElement("section"),
        img = target.createElement("img");
      const name = "drawingPage" + target.body.children.length;
      style.textContent += `@page ${name}{size:${width}pt ${height}pt;margin:0}`;
      sheet.className = "sheet";
      sheet.style.cssText = `page:${name};width:${width}pt;height:${height}pt;`;
      const swapped = imageRotation % 180 !== 0;
      img.style.cssText = `width:${swapped ? height : width}pt;height:${swapped ? width : height}pt;transform:rotate(${imageRotation}deg);`;
      img.alt = title;
      sheet.appendChild(img);
      target.body.appendChild(sheet);
      const ready = imageReady(img, signal);
      img.src = src;
      return ready;
    };

    if (pdf) {
      if (
        pageNumber !== undefined &&
        (!Number.isInteger(pageNumber) ||
          pageNumber < 1 ||
          pageNumber > pdf.numPages)
      )
        throw Error("打印页码无效");
      for (
        let number = pageNumber || 1;
        number <= (pageNumber || pdf.numPages);
        number++
      ) {
        signal.throwIfAborted();
        const page = await pdf.getPage(number);
        signal.throwIfAborted();
        const angle = (page.rotate + rotation) % 360;
        const original = page.getViewport({ scale: 1, rotation: angle });
        // 200 DPI for normal paper; bound the temporary canvas for large drawings.
        const scale = Math.min(
          200 / 72,
          Math.sqrt(16000000 / (original.width * original.height)),
        );
        const viewport = page.getViewport({ scale, rotation: angle });
        const canvas = target.createElement("canvas");
        canvas.width = Math.ceil(viewport.width);
        canvas.height = Math.ceil(viewport.height);
        const context = canvas.getContext("2d");
        if (!context) throw new Error("浏览器无法准备打印图纸");
        const task = page.render({
          canvas,
          canvasContext: context,
          viewport,
          intent: "print",
        });
        const abort = () => task.cancel();
        signal.addEventListener("abort", abort, { once: true });
        try {
          signal.throwIfAborted();
          await task.promise;
          signal.throwIfAborted();
          const file = await new Promise<Blob>((resolve, reject) =>
            canvas.toBlob(
              (value) =>
                value ? resolve(value) : reject(new Error("打印页面生成失败")),
              "image/png",
            ),
          );
          signal.throwIfAborted();
          const url = URL.createObjectURL(file);
          urls.push(url);
          await appendPage(url, original.width, original.height);
        } finally {
          signal.removeEventListener("abort", abort);
          canvas.width = canvas.height = 0;
        }
      }
    } else {
      const image = target.createElement("img");
      const loaded = imageReady(image, signal);
      image.src = imageUrl!;
      await loaded;
      const swapped = rotation % 180 !== 0;
      const landscape = swapped
        ? image.naturalHeight > image.naturalWidth
        : image.naturalWidth > image.naturalHeight;
      await appendPage(
        imageUrl!,
        landscape ? 841.89 : 595.28,
        landscape ? 595.28 : 841.89,
        rotation,
      );
    }
    signal.throwIfAborted();
    await new Promise<void>((resolve, reject) => {
      let timer: ReturnType<typeof setTimeout>;
      const finish = () => {
        cleanup();
        resolve();
      };
      const abort = () => {
        cleanup();
        reject(signal.reason || new DOMException("取消打印", "AbortError"));
      };
      const cleanup = () => {
        clearTimeout(timer);
        view.removeEventListener("afterprint", finish);
        signal.removeEventListener("abort", abort);
      };
      view.addEventListener("afterprint", finish, { once: true });
      signal.addEventListener("abort", abort, { once: true });
      // Safety cleanup for browsers which do not emit afterprint after cancellation.
      timer = setTimeout(finish, 5 * 60 * 1000);
      try {
        view.focus();
        view.print();
      } catch {
        cleanup();
        reject(new Error("浏览器未能打开打印，请下载原文件后打印"));
      }
    });
  } finally {
    frame.remove();
    urls.forEach((url) => URL.revokeObjectURL(url));
  }
}

function imageReady(image: HTMLImageElement, signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    const cleanup = () => {
      clearTimeout(timer);
      image.onload = image.onerror = null;
      signal.removeEventListener("abort", abort);
    };
    const abort = () => {
      cleanup();
      reject(signal.reason || new DOMException("取消打印", "AbortError"));
    };
    const timer = setTimeout(() => {
      cleanup();
      reject(new Error("打印图纸读取超时，请重试"));
    }, 30000);
    image.onload = () => {
      cleanup();
      resolve();
    };
    image.onerror = () => {
      cleanup();
      reject(new Error("打印图纸读取失败，请重试"));
    };
    signal.addEventListener("abort", abort, { once: true });
    if (signal.aborted) abort();
  });
}
