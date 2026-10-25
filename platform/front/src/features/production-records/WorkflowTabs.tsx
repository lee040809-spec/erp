import type { WorkflowStep } from "../../utils/partWorkflow";
const main: [WorkflowStep, string][] = [
  ["process", "1 安排与加工"],
  ["completion", "2 确认合格"],
  ["delivery", "3 登记交货"],
];
const more: [WorkflowStep, string][] = [
  ["batches", "分批计划"],
  ["outsourcing", "外协"],
  ["records", "报废 / 异常 / 记录"],
];
export default function WorkflowTabs({
  step,
  busy,
  navigate,
}: {
  step: WorkflowStep;
  busy: boolean;
  navigate: (step: WorkflowStep) => void;
}) {
  const render = ([key, label]: [WorkflowStep, string]) => (
    <button
      key={key}
      type="button"
      aria-current={step === key ? "step" : undefined}
      disabled={busy}
      className={`erp-btn ${step === key ? "erp-primary" : ""}`}
      onClick={() => navigate(key)}
    >
      {label}
    </button>
  );
  return (
    <nav aria-label="零件操作流程" className="erp-workflow-tabs">
      {main.map(render)}
      <details open={more.some(([key]) => key === step)} className="flex-1">
        <summary className="erp-btn cursor-pointer w-fit">更多业务</summary>
        <div className="flex flex-wrap gap-2 mt-2">{more.map(render)}</div>
      </details>
    </nav>
  );
}
