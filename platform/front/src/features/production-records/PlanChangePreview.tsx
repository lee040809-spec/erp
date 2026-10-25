import { processDate } from "../../utils/processDates";
export default function PlanChangePreview({
  before,
  draft,
  dueDate,
}: {
  before: { planned_start_at?: string | null; planned_end_at?: string | null };
  draft: { planned_start_at: string; planned_end_at: string };
  dueDate?: string | null;
}) {
  const oldStart = processDate(before.planned_start_at),
    oldEnd = processDate(before.planned_end_at),
    due = processDate(dueDate);
  if (oldStart === draft.planned_start_at && oldEnd === draft.planned_end_at)
    return null;
  const days =
    draft.planned_end_at && due
      ? Math.round(
          (Date.parse(draft.planned_end_at + "T00:00:00Z") -
            Date.parse(due + "T00:00:00Z")) /
            86400000,
        )
      : null;
  return (
    <aside
      className={`rounded-lg p-3 text-sm ${days !== null && days > 0 ? "bg-amber-50 text-amber-900" : "bg-blue-50 text-blue-900"}`}
      aria-label="计划修改预览"
    >
      <strong>保存前核对本工序新计划</strong>
      <p className="mt-1">
        原计划：{oldStart || "未安排"} → {oldEnd || "未安排"}
      </p>
      <p>
        新计划：{draft.planned_start_at || "未安排"} →{" "}
        {draft.planned_end_at || "未安排"}
      </p>
      <p>
        客户交期：{due || "未填写"}
        {days !== null && Number.isFinite(days)
          ? days > 0
            ? `；新结束日期晚于客户交期 ${days} 天`
            : "；新结束日期在客户交期以内"
          : ""}
      </p>
      <p className="text-xs mt-1">
        本次只调整该工序，请一并核对后续工序的衔接。
      </p>
    </aside>
  );
}
