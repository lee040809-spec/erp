const fs = require('fs'), path = require('path'), vm = require('vm'), assert = require('assert/strict');
const root = path.resolve(__dirname, '..'), ts = require(root + '/node_modules/typescript'), React = require(root + '/node_modules/react');
let current, allowDiscard = false, confirms = 0, reads = 0, writes = 0, failDrawingRead = false, resolveWrite;
const modules = new Map(), stubs = new Map();
const react = { ...React,
  useState(initial) { const c = current, i = c.s++; if (!(i in c.states)) c.states[i] = typeof initial === 'function' ? initial() : initial; return [c.states[i], value => c.states[i] = typeof value === 'function' ? value(c.states[i]) : value]; },
  useRef(initial) { const c = current, i = c.r++; return c.refs[i] || (c.refs[i] = { current: initial }); },
  useMemo(fn, deps) { const c = current, i = c.m++, previous = c.memos[i]; if (!previous || !same(previous.deps, deps)) c.memos[i] = { value: fn(), deps }; return c.memos[i].value; },
  useCallback(fn, deps) { return react.useMemo(() => fn, deps); },
  useEffect(fn, deps) { const c = current, i = c.e++; if (!same(c.deps[i], deps)) { c.deps[i] = deps; c.effects.push(() => { c.cleanups[i]?.(); c.cleanups[i] = fn(); }); } },
};
function same(a, b) { return a && b && a.length === b.length && a.every((v, i) => Object.is(v, b[i])); }
function runner(fn) { const c = { states: [], refs: [], memos: [], deps: [], cleanups: [] }; return { render() { current = c; c.s = c.r = c.m = c.e = 0; c.effects = []; return fn(); }, effects() { c.effects.forEach(f => f()); }, close() { c.cleanups.forEach(f => f?.()); } }; }
const drawing = { id: 1, is_current: true, file_name: 'fixture.pdf', size: 10, created_at: '2026-10-14', mime_type: 'application/pdf', version: 'A' };
function load(file) {
  if (modules.has(file)) return modules.get(file);
  const exports = {}, code = ts.transpileModule(fs.readFileSync(root + '/src/' + file, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true } }).outputText;
  vm.runInNewContext(code, { exports, Error, Date, Math, Number, Intl, document: { body: {} }, window: { confirm() { confirms++; return allowDiscard; } }, require(name) {
    if (name === 'react') return react;
    if (name === 'react/jsx-runtime') return require(root + '/node_modules/react/jsx-runtime');
    if (name === 'react-dom') return { createPortal: node => node };
    if (name === 'lucide-react') return require(root + '/node_modules/lucide-react');
    if (name.endsWith('.css')) return {};
    if (name.endsWith('/useDraftGuard')) return load('hooks/useDraftGuard.ts');
    if (name.endsWith('/useAsyncResource')) return load('hooks/useAsyncResource.ts');
    if (name.endsWith('/money')) return load('utils/money.ts');
    if (name.endsWith('/orderAmounts')) return load('utils/orderAmounts.ts');
    if (name.endsWith('/productionFlow')) return { flowRequest: async (url, method, payload) => { if (method) { writes++; await new Promise(done => { resolveWrite = done; }); return payload; } reads++; return { items: [{ id: 7, name: '供应商测试', active: true }], page: 1, has_more: false, can_manage: true }; } };
    if (name === './api') return { listDrawings: async () => { if (failDrawingRead) throw Error('清单连接失败'); return { drawings: [drawing], can_manage: true, legacy: false, events: [] }; }, setDrawingCurrent: async () => {}, drawingRequest: async () => ({}) };
    if (name.endsWith('/shared')) return { StatusBadge: () => null, formatDate: value => value || '-' };
    const key = name.split('/').pop(); if (!stubs.has(key)) stubs.set(key, () => null);
    return { __esModule: true, default: stubs.get(key), invalidDeliveryPlan: () => false, invalidOrderProcessDates: () => false };
  } }); modules.set(file, exports); return exports;
}
function nodes(tree, predicate, result = []) { if (Array.isArray(tree)) tree.forEach(n => nodes(n, predicate, result)); else if (tree?.props) { if (predicate(tree)) result.push(tree); nodes(tree.props.children, predicate, result); } return result; }
function text(tree) { return typeof tree === 'string' || typeof tree === 'number' ? String(tree) : Array.isArray(tree) ? tree.map(text).join('') : tree?.props ? text(tree.props.children) : ''; }
const flush = () => new Promise(done => setImmediate(done));
async function run() {
  const Guard = load('hooks/useDraftGuard.ts').useDraftGuard; let saving = false;
  const guardRun = runner(() => Guard(true, 'new', saving)); let guard = guardRun.render(); guardRun.effects(); assert.equal(guard.confirmDiscard(), true);
  guard.markChanged(); assert.equal(guard.confirmDiscard(), false); allowDiscard = true; assert.equal(guard.confirmDiscard(), true);
  saving = true; guard = guardRun.render(); const before = confirms; assert.equal(guard.confirmDiscard(), false); assert.equal(confirms, before); saving = false;
  const Resource = load('hooks/useAsyncResource.ts').useAsyncResource; let calls = 0, fail = false;
  const request = async () => { calls++; if (fail) throw Error('读取中断'); return { value: calls }; };
  const resourceRun = runner(() => Resource(request, { value: 0 })); resourceRun.render(); resourceRun.effects(); await flush(); let resource = resourceRun.render(); assert.equal(resource.data.value, 1);
  fail = true; assert.equal(await resource.reload(), false); resource = resourceRun.render(); assert.equal(resource.data.value, 1); assert.equal(resource.error, '读取中断');
  fail = false; assert.equal(await resource.reload(), true); resource = resourceRun.render(); assert.equal(resource.data.value, 3);
  const pending = []; const delayed = () => new Promise(resolve => pending.push(resolve)); const race = runner(() => Resource(delayed, 'empty'));
  let racing = race.render(); race.effects(); const next = racing.reload(); pending[1]('latest'); await next; pending[0]('obsolete'); await flush(); assert.equal(race.render().data, 'latest'); race.close();
  const Suppliers = load('pages/OutsourcingSuppliers.tsx').default; const supplierRun = runner(() => Suppliers({})); supplierRun.render(); supplierRun.effects(); await flush(); let tree = supplierRun.render();
  const beforeReads = reads; nodes(tree, n => n.type === 'form' && n.props.role === 'search')[0].props.onSubmit({ preventDefault() {} }); await flush(); assert.equal(reads, beforeReads + 1, 'Same query must actually reload');
  tree = supplierRun.render(); nodes(tree, n => n.type === 'button' && text(n) === '编辑 / 启停')[0].props.onClick(); tree = supplierRun.render(); supplierRun.effects();
  const firstEditor = nodes(tree, n => n.type === stubs.get('SupplierEditor'))[0]; firstEditor.props.onDirty(); allowDiscard = true;
  nodes(tree, n => n.type === 'button' && text(n) === '编辑 / 启停')[0].props.onClick(); tree = supplierRun.render();
  assert.notEqual(nodes(tree, n => n.type === stubs.get('SupplierEditor'))[0].key, firstEditor.key, 'Discard and reopen same supplier must reset the form');
  const Editor = load('components/SupplierEditor.tsx').default; let saved = 0, busy = [];
  const editorRun = runner(() => Editor({ supplier: { id: 7, name: ' 测试供应商 ', active: true, contact_name: '', phone: '', address: '', capabilities: '', notes: '' }, onDirty() {}, onBusy: v => busy.push(v), onCancel() {}, onSaved: async () => saved++ })); tree = editorRun.render();
  const form = nodes(tree, n => n.type === 'form')[0]; const savingRequest = form.props.onSubmit({ preventDefault() {} }); form.props.onSubmit({ preventDefault() {} }); assert.equal(writes, 1); assert.equal(nodes(editorRun.render(), n => n.type === 'fieldset')[0].props.disabled, true); resolveWrite(); await savingRequest; assert.equal(saved, 1); assert.deepEqual(busy, [true, false]);
  const Modal = load('components/OrderModal.tsx').default; let order = { customer_id: 9, contact_name: '旧联系人', total_amount: 1, items: [{ part_name: '测试零件', quantity: 1, production_quantity: 1, unit_price: 1, processes: [] }] }, closed = 0;
  const props = { show: true, onClose: () => closed++, customers: [{ id: 9, name: '完整企业客户名称', short_name: '客户简称' }], customerSearch: '完整企业', setCustomerSearch() {}, showCustomerDropdown: true, setShowCustomerDropdown() {}, customerDropdownRef: { current: null }, formErrors: {}, setFormErrors() {}, isSaving: false, onSubmit() {}, setNewOrder: v => order = v };
  const orderRun = runner(() => Modal({ ...props, newOrder: order })); tree = orderRun.render(); orderRun.effects(); assert.ok(text(tree).includes('客户简称'), 'Full-name query matches customer with a short name');
  nodes(tree, n => n.type === 'input' && n.props['aria-label'] === '选择客户')[0].props.onChange({ target: { value: '另一个客户' } }); assert.equal(order.customer_id, undefined); assert.equal(order.contact_name, undefined);
  allowDiscard = false; tree = orderRun.render(); nodes(tree, n => n.type === 'button' && n.props['aria-label'] === '关闭订单编辑')[0].props.onClick(); assert.equal(closed, 0);
  nodes(tree, n => n.type === 'button' && n.props['aria-label'] === '删除零件1')[0].props.onClick(); assert.equal(order.items.length, 1);
  allowDiscard = true; nodes(tree, n => n.type === 'button' && n.props['aria-label'] === '关闭订单编辑')[0].props.onClick(); assert.equal(closed, 1);
  props.isSaving = true; tree = orderRun.render(); assert.equal(nodes(tree, n => n.type === 'fieldset')[0].props.disabled, true);
  const Manager = load('features/drawings/DrawingManager.tsx').default; const drawingRun = runner(() => Manager({ itemId: 1, partName: '测试零件', onClose() {} })); drawingRun.render(); drawingRun.effects(); await flush(); tree = drawingRun.render();
  const upload = nodes(tree, n => n.type === stubs.get('DrawingUpload'))[0]; assert.ok(upload); failDrawingRead = true; await upload.props.onUploaded(drawing); tree = drawingRun.render(); assert.ok(text(tree).includes('图纸已保存，但清单刷新失败')); assert.ok(text(tree).includes('不要重复上传')); assert.ok(!nodes(tree, n => n.type === stubs.get('DrawingUpload')).length, 'Write controls hidden when metadata refresh failed');
  const Pagination = load('components/Pagination.tsx').default; tree = Pagination({ total: 1000000000, page: 5000000, pageSize: 10, disabled: true, onPageChange() {}, onPageSizeChange() {} }); assert.ok(nodes(tree, n => n.type === 'button').length <= 7); assert.ok(nodes(tree, n => n.type === 'button').every(n => n.props.disabled));
  console.log('PASS: unsaved discard and busy guards, retained data and stale-response isolation, repeat supplier query, single guarded supplier save, full-name customer search and old-customer clearing, order removal confirmation, drawing saved/refresh-failed feedback, bounded busy pagination. No business database writes.');
}
run().catch(error => { console.error(error); process.exitCode = 1; });
