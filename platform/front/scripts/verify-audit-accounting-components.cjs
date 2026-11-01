const assert = require("assert/strict");
const h = require("./verify-accounting-flows.cjs");
const event = { preventDefault() {} };

async function run() {
  // Execute the actual target components and async hook; only transport/layout
  // dependencies are mocked. Nothing contacts a database or saves ERP records.
  h.realComponents.add("DeliveryReconciliation");
  h.realComponents.add("CostCompletionPicker");
  const Reconciliation = h.load(
    "features/accounting/DeliveryReconciliation.tsx",
  ).default;
  const r = h.runner(() => Reconciliation({}));
  r.render();
  let tree = await h.settle(r);
  h.nodes(
    tree,
    (n) => n.type === "input" && n.props.type === "checkbox",
  )[0].props.onChange();
  tree = r.render();
  h.button(tree, "下一页").props.onClick();
  r.render();
  tree = await h.settle(r);
  h.nodes(
    tree,
    (n) => n.type === "input" && n.props.type === "checkbox",
  )[0].props.onChange();
  tree = r.render();
  assert.ok(
    /已选 2/.test(h.text(tree)),
    "selected deliveries persist across pages",
  );
  h.field(tree, "备注").props.onChange({ target: { value: "保留备注" } });
  tree = r.render();
  const before = h.writes.length;
  h.configure({ holdWrites: true, failReconciliation: true });
  const save = h.button(tree, "保存对账并标记已对账").props.onClick();
  h.button(tree, "保存对账并标记已对账").props.onClick();
  assert.equal(h.writes.length, before + 1, "double click must make one write");
  h.releases.shift()();
  await save;
  h.configure({ holdWrites: false });
  tree = r.render();
  assert.ok(h.text(tree).includes("填写内容已保留"));
  assert.equal(h.field(tree, "备注").props.value, "保留备注");
  const failed = h.writes.at(-1).payload;
  h.configure({ failReconciliation: false });
  await h.button(tree, "保存对账并标记已对账").props.onClick();
  tree = await h.settle(r);
  const retried = h.writes.at(-1).payload;
  assert.equal(
    retried.request_id,
    failed.request_id,
    "identical retry keeps idempotency key",
  );
  assert.deepEqual(Array.from(retried.delivery_ids), [1, 2]);
  assert.ok(h.text(tree).includes("已保存对账单 TEST-DZ"));
  assert.equal(
    h.button(tree, "保存对账并标记已对账"),
    undefined,
    "success clears selection",
  );
  r.close();

  const Picker = h.load("features/accounting/CostCompletionPicker.tsx").default;
  let picked = "",
    month = "";
  const p = h.runner(() =>
    Picker({
      itemId: 111,
      value: picked,
      onChange: (id, m) => {
        picked = id;
        month = m;
      },
    }),
  );
  p.render();
  tree = await h.settle(p);
  h.nodes(tree, (n) => n.type === "select")[0].props.onChange({
    target: { value: "8" },
  });
  assert.equal(picked, "8");
  assert.equal(month, "2025-09");
  tree = p.render();
  h.button(tree, "较早批次").props.onClick();
  assert.equal(picked, "");
  p.render();
  tree = await h.settle(p);
  h.configure({ failCompletionRead: true });
  h.button(tree, "较早批次").props.onClick();
  p.render();
  tree = await h.settle(p);
  assert.equal(
    h.nodes(tree, (n) => n.type === "select")[0].props.disabled,
    true,
  );
  h.configure({ failCompletionRead: false });
  await h.button(tree, "重试").props.onClick();
  tree = p.render();
  assert.equal(
    h.nodes(tree, (n) => n.type === "select")[0].props.disabled,
    false,
  );
  p.close();

  const Panel = h.load("components/FinanceReviewPanel.tsx").default;
  for (const status of ["pending", "priced"]) {
    h.configure({ historyPricing: status });
    h.candidate.unit_price = status === "pending" ? null : "3.20";
    const panel = h.runner(() =>
      Panel({
        year: "2026",
        month: "2026-01",
        selectionVersion: 1,
        initialTab: "history",
        access: { can_import_history: true, can_manage_costs: false },
        onImported() {},
      }),
    );
    panel.render();
    tree = await h.settle(panel);
    h.button(tree, "补录完工").props.onClick();
    tree = panel.render();
    const dialog = h.nodes(
      tree,
      (n) => n.type === h.stubs.get("HistoryCompletionDialog"),
    )[0];
    await dialog.props.onSubmit(event);
    tree = panel.render();
    assert.ok(
      h.text(tree).includes(status === "pending" ? "金额待补价" : "已计入产值"),
    );
    if (status === "pending")
      assert.equal(h.writes.at(-1).payload.expected_price, "0");
    panel.close();
  }
  console.log(
    "PASS: actual delivery reconciliation cross-page selection, duplicate guard, failed-save retention, identical retry and success cleanup; actual completion picker paging/read failure/retry; history feedback uses server pricing status. All transport mocked.",
  );
}
run().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
