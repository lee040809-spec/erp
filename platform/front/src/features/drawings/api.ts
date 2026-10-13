import { authFetch } from "../../components/shared";
export interface PartDrawing {
  id: number;
  order_item_id?: number;
  source_drawing_id?: number;
  replaces_id?: number;
  file_name: string;
  mime_type: string;
  size: number;
  drawing_number: string;
  version: string;
  notes: string;
  is_current: boolean;
  uploaded_by: number;
  created_at: string;
  part_name?: string;
  part_number?: string;
  order_number?: string;
  customer_name?: string;
}
export interface DrawingList {
  drawings: PartDrawing[];
  legacy: boolean;
  can_manage: boolean;
  customer_id?: number;
  events: {
    id: number;
    drawing_id: number;
    action: string;
    reason: string;
    actor_id: number;
    created_at: string;
  }[];
}
export const fileAccept = ".pdf,.jpg,.jpeg,.png,.dwg,.dxf,.step,.stp";
export const maxDrawingSize = 20 * 1024 * 1024;
export async function drawingRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const response = await authFetch("/api/platform/" + path, init);
  const body = await response.json();
  if (!response.ok)
    throw new Error(body.error || body.message || "图纸操作失败");
  return body.data as T;
}
export function listDrawings(itemId: number) {
  return drawingRequest<DrawingList>(`order-items/${itemId}/drawings`);
}
export function uploadDrawing(file: File, fields: Record<string, string>) {
  const form = new FormData();
  for (const [key, value] of Object.entries(fields)) form.append(key, value);
  form.append("file", file);
  return drawingRequest<PartDrawing>("drawings", {
    method: "POST",
    body: form,
  });
}
export function referenceDrawing(
  sourceId: number,
  itemId?: number,
  reason = "",
) {
  return drawingRequest<PartDrawing>("drawings/reference", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ source_id: sourceId, item_id: itemId || 0, reason }),
  });
}
export function setDrawingCurrent(
  id: number,
  isCurrent: boolean,
  reason: string,
) {
  return drawingRequest("drawings/" + id, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ is_current: isCurrent, reason }),
  });
}
export async function drawingBlob(id: number, signal?: AbortSignal) {
  const response = await authFetch(`/api/platform/drawings/${id}/file`, {
    signal,
  });
  if (!response.ok) {
    const data = await response.json();
    throw new Error(data.error || "图纸读取失败");
  }
  return response.blob();
}
