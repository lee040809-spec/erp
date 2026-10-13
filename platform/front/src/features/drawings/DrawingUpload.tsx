import { useRef, useState } from "react";
import { fileAccept, maxDrawingSize, PartDrawing, uploadDrawing } from "./api";
export default function DrawingUpload({
  itemId,
  drawings,
  reason,
  disabled,
  onBusy,
  onUploaded,
  onError,
}: {
  itemId?: number;
  drawings: PartDrawing[];
  reason: string;
  disabled: boolean;
  onBusy: (busy: boolean) => void;
  onUploaded: (drawing: PartDrawing) => Promise<void>;
  onError: (error: string) => void;
}) {
  const input = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null),
    [version, setVersion] = useState(""),
    [number, setNumber] = useState(""),
    [notes, setNotes] = useState(""),
    [replacement, setReplacement] = useState("");
  const submit = async () => {
    if (!file) return;
    if (file.size > maxDrawingSize || file.size === 0) {
      onError("请选择不超过20MB的非空图纸");
      return;
    }
    onBusy(true);
    onError("");
    try {
      const drawing = await uploadDrawing(file, {
        item_id: String(itemId || 0),
        version,
        drawing_number: number,
        notes,
        replaces_id: replacement,
        reason,
      });
      await onUploaded(drawing);
      setFile(null);
      setVersion("");
      setNumber("");
      setNotes("");
      setReplacement("");
      if (input.current) input.current.value = "";
    } catch (e) {
      onError(e instanceof Error ? e.message : "图纸上传失败");
    } finally {
      onBusy(false);
    }
  };
  return (
    <div className="drawing-upload">
      <div className="drawing-manager-fields">
        <label className="erp-label">
          图纸文件
          <input
            ref={input}
            type="file"
            accept={fileAccept}
            disabled={disabled}
            onChange={(e) => setFile(e.target.files?.[0] || null)}
          />
        </label>
        <label className="erp-label">
          图纸编号（选填）
          <input
            className="erp-input"
            maxLength={100}
            value={number}
            disabled={disabled}
            onChange={(e) => setNumber(e.target.value)}
          />
        </label>
        <label className="erp-label">
          版本（选填）
          <input
            className="erp-input"
            maxLength={100}
            placeholder="如 A、V1.1"
            value={version}
            disabled={disabled}
            onChange={(e) => setVersion(e.target.value)}
          />
        </label>
        <label className="erp-label">
          备注（选填）
          <input
            className="erp-input"
            maxLength={500}
            value={notes}
            disabled={disabled}
            onChange={(e) => setNotes(e.target.value)}
          />
        </label>
        {!!itemId && (
          <label className="erp-label">
            替换已有版本
            <select
              className="erp-input"
              value={replacement}
              disabled={disabled}
              onChange={(e) => setReplacement(e.target.value)}
            >
              <option value="">新增一份图纸</option>
              {drawings
                .filter((d) => d.is_current)
                .map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.file_name}
                    {d.version ? ` · ${d.version}` : ""}
                  </option>
                ))}
            </select>
          </label>
        )}
      </div>
      <div className="flex gap-3 items-center flex-wrap mt-3">
        <button
          type="button"
          className="erp-btn erp-primary"
          disabled={disabled || !file}
          onClick={() => void submit()}
        >
          {disabled ? "处理中…" : "上传图纸"}
        </button>
        <span className="text-xs text-slate-500">
          PDF / JPG / PNG可预览；CAD可下载。每份不超过20MB。
        </span>
      </div>
    </div>
  );
}
