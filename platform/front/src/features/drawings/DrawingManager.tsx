import { useCallback, useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import type { DrawingButtonProps } from "./DrawingButton";
import {
  DrawingList,
  PartDrawing,
  drawingRequest,
  listDrawings,
  setDrawingCurrent,
} from "./api";
import DrawingUpload from "./DrawingUpload";
import DrawingHistoryPicker from "./DrawingHistoryPicker";
import DrawingPreview from "./DrawingPreview";
import { useAsyncResource } from "../../hooks/useAsyncResource";
import "./drawings.css";
const initial: DrawingList = {
  drawings: [],
  legacy: false,
  can_manage: false,
  events: [],
};
export default function DrawingManager(
  p: DrawingButtonProps & {
    onClose: () => void;
    onCountChange?: (count: number) => void;
  },
) {
  const request = useCallback(() => listDrawings(p.itemId!), [p.itemId]);
  const {
    data,
    loading,
    error: readError,
    reload: load,
  } = useAsyncResource(request, initial, !!p.itemId);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [reason, setReason] = useState(""),
    [history, setHistory] = useState(false),
    [preview, setPreview] = useState<PartDrawing | null>(null),
    [legacy, setLegacy] = useState("");
  useEffect(() => {
    if (p.itemId && !loading && !readError)
      p.onCountChange?.(
        data.drawings.filter((d) => d.is_current).length +
          (data.legacy ? 1 : 0),
      );
  }, [p.itemId, p.onCountChange, data, loading, readError]);
  const rows = p.itemId ? data.drawings : p.drafts || [];
  const canManage =
    !p.disabled &&
    !loading &&
    !readError &&
    (p.itemId ? data.can_manage : !!p.onDraftChange);
  const added = async (d: PartDrawing) => {
    let refreshed = true;
    if (p.itemId) {
      refreshed = await load();
      p.onChanged?.();
    } else p.onDraftChange?.([...rows, d]);
    setNotice(
      p.itemId
        ? refreshed
          ? "图纸已保存，并关联当前零件。"
          : "图纸已保存，但清单刷新失败。请重新读取清单，不要重复上传。"
        : "图纸已上传，保存订单后完成零件关联。",
    );
    setReason("");
  };
  const state = async (d: PartDrawing) => {
    if (
      !window.confirm(
        d.is_current
          ? "移至历史后保留原文件和记录，但不再作为当前生产图纸。继续吗？"
          : "将此历史版本恢复为当前生产图纸，继续吗？",
      )
    )
      return;
    setBusy(true);
    setError("");
    try {
      await setDrawingCurrent(d.id, !d.is_current, reason);
      const refreshed = await load();
      p.onChanged?.();
      setNotice(
        refreshed
          ? "图纸版本状态已更新。"
          : "版本修改已保存，但清单刷新失败。请重新读取清单，不要重复操作。",
      );
      setReason("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "修改失败");
    } finally {
      setBusy(false);
    }
  };
  const showLegacy = async () => {
    setBusy(true);
    setError("");
    try {
      const result = await drawingRequest<{ drawing_data?: string }>(
        `order-items/${p.itemId}/drawing`,
      );
      if (!result.drawing_data) throw new Error("原有图纸不存在");
      if (
        !result.drawing_data.startsWith("data:image/") &&
        !result.drawing_data.startsWith("data:application/pdf;")
      )
        throw new Error("原有图纸格式无法在线预览，请联系管理员核对");
      setLegacy(result.drawing_data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "读取失败");
    } finally {
      setBusy(false);
    }
  };
  const labels: Record<string, string> = {
    linked: "关联图纸",
    replaced: "替换旧版本",
    archived: "移至历史",
    current: "设为当前",
    legacy_import: "旧图纸迁移",
  };
  return createPortal(
    <div data-escape-dialog data-escape-blocked={busy}
      className="erp-overlay drawing-manager-overlay"
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-label="零件图纸"
        className="drawing-manager"
      >
        <header>
          <div>
            <h3 className="font-bold">{p.partName || "新零件"} · 图纸</h3>
            <p className="text-xs text-slate-500 mt-1">
              {p.partNumber && `${p.partNumber} · `}
              当前生产图纸与历史版本统一维护
            </p>
          </div>
          <button data-escape-close
            type="button"
            aria-label="关闭零件图纸"
            className="erp-btn"
            disabled={busy}
            onClick={p.onClose}
          >
            <X size={18} />
          </button>
        </header>
        <div className="drawing-manager-scroll">
          {loading && <p role="status">正在读取图纸信息…</p>}
          {error && (
            <p role="alert" className="text-sm text-rose-600 mb-3">
              {error}
            </p>
          )}
          {readError && (
            <p role="alert" className="text-sm text-amber-700 mb-3">
              清单读取失败：{readError}
              <button
                type="button"
                className="erp-btn ml-2"
                disabled={busy}
                onClick={() => void load()}
              >
                重试读取
              </button>
            </p>
          )}
          {notice && (
            <p role="status" className="text-sm text-emerald-700 mb-3">
              {notice}
            </p>
          )}
          {canManage && (
            <>
              <label className="erp-label">
                图纸变更说明
                <input
                  className="erp-input"
                  maxLength={500}
                  value={reason}
                  disabled={busy}
                  placeholder="已安排工序或投入生产时必填；请核对工艺与加工要求"
                  onChange={(e) => setReason(e.target.value)}
                />
              </label>
              <DrawingUpload
                itemId={p.itemId}
                drawings={rows}
                reason={reason}
                disabled={busy}
                onBusy={setBusy}
                onUploaded={added}
                onError={setError}
              />
            </>
          )}
          <div className="flex justify-between items-center">
            <strong className="text-sm">
              {p.itemId ? "生产图纸" : "待保存到订单的图纸"}{" "}
              {loading
                ? "读取中…"
                : readError
                  ? "未读取"
                  : rows.filter((d) => d.is_current || !p.itemId).length +
                    (data.legacy ? 1 : 0)}
            </strong>
            {p.itemId && rows.some((d) => !d.is_current) && (
              <button
                type="button"
                className="erp-btn"
                onClick={() => setHistory(!history)}
              >
                {history ? "收起历史版本" : "查看历史版本"}
              </button>
            )}
          </div>
          {!rows.length && !loading && !readError && !data.legacy && (
            <p className="text-sm text-slate-500 my-4">
              暂无图纸{canManage ? "，可以上传或引用历史图纸。" : "。"}
            </p>
          )}
          {rows
            .filter((d) => !p.itemId || d.is_current || history)
            .map((d) => (
              <div className="drawing-row" key={d.id}>
                <div>
                  <strong>
                    {d.file_name}
                    <span
                      className={`ml-2 text-xs ${d.is_current ? "text-emerald-700" : "text-slate-500"}`}
                    >
                      {p.itemId
                        ? d.is_current
                          ? "当前生产"
                          : "历史版本"
                        : "待关联"}
                    </span>
                  </strong>
                  <small>
                    {d.drawing_number && `图号 ${d.drawing_number} · `}版本{" "}
                    {d.version || "未填写"} ·{" "}
                    {(d.size / 1024 / 1024).toFixed(2)} MB ·{" "}
                    {new Date(d.created_at).toLocaleDateString("zh-CN")}
                  </small>
                  {d.notes && <small>{d.notes}</small>}
                </div>
                <div className="drawing-row-actions">
                  <button
                    type="button"
                    className="erp-btn"
                    disabled={busy}
                    onClick={() => setPreview(d)}
                  >
                    预览 / 下载
                  </button>
                  {canManage && p.itemId && (
                    <button
                      type="button"
                      className="erp-btn"
                      disabled={busy}
                      onClick={() => void state(d)}
                    >
                      {d.is_current ? "移至历史" : "恢复当前"}
                    </button>
                  )}
                  {!p.itemId && p.onDraftChange && (
                    <button
                      type="button"
                      className="erp-btn"
                      disabled={busy}
                      onClick={() =>
                        p.onDraftChange?.(rows.filter((r) => r.id !== d.id))
                      }
                    >
                      取消关联
                    </button>
                  )}
                </div>
              </div>
            ))}
          {data.legacy && (
            <div className="drawing-row">
              <span>原有图纸</span>
              <button
                type="button"
                className="erp-btn"
                disabled={busy}
                onClick={() => void showLegacy()}
              >
                预览 / 下载
              </button>
            </div>
          )}
          {canManage && (
            <div className="mt-4">
              <DrawingHistoryPicker
                itemId={p.itemId}
                customerId={p.customerId || data.customer_id}
                reason={reason}
                disabled={busy}
                onBusy={setBusy}
                onAdded={added}
                onPreview={setPreview}
                onError={setError}
              />
            </div>
          )}
          {!!data.events.length && (
            <details className="mt-4 text-xs text-slate-500">
              <summary className="cursor-pointer">最近变更记录</summary>
              {data.events.map((e) => (
                <p className="mt-2" key={e.id}>
                  {new Date(e.created_at).toLocaleString("zh-CN")} ·{" "}
                  {labels[e.action] || e.action} · 操作人 #{e.actor_id}
                  {e.reason && ` · ${e.reason}`}
                </p>
              ))}
            </details>
          )}
        </div>
      </section>
      {(preview || legacy) && (
        <DrawingPreview
          drawing={preview || undefined}
          legacy={legacy || undefined}
          onClose={() => {
            setPreview(null);
            setLegacy("");
          }}
        />
      )}
    </div>,
    document.body,
  );
}
