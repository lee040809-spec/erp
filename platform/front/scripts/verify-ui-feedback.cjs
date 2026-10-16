const fs = require('fs'), path = require('path'), vm = require('vm'), assert = require('assert/strict');
const root = path.resolve(__dirname, '..'), ts = require(root + '/node_modules/typescript'), React = require(root + '/node_modules/react');
let state = [], deps = [], effects = [], refs = [], index = 0, refIndex = 0, effectIndex = 0, failPlan = true, failFinance = false, requests = [], saved = 0, sent;
const react = { ...React, useMemo: f => f(), useCallback: f => f, useRef(v) { const i = refIndex++; return refs[i] || (refs[i] = { current: v }); }, useState(v) { const i = index++; if (!(i in state)) state[i] = v; return [state[i], v => state[i] = typeof v === 'function' ? v(state[i]) : v]; }, useEffect(f, next) { const i = effectIndex++; if (!deps[i] || next.some((v, j) => v !== deps[i][j])) { deps[i] = next; effects.push(f); } } };
const storage = new Map([['auth_token', 'test-token']]), location = { href: '/platform/' };
let response = { status: 200 };
function load(file) {
  const exports = {};
  const code = ts.transpileModule(fs.readFileSync(root + '/src/' + file, 'utf8'), { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, jsx: ts.JsxEmit.ReactJSX, esModuleInterop: true } }).outputText;
  vm.runInNewContext(code, { exports, Error, Headers, window: { location }, localStorage: { getItem: k => storage.get(k), removeItem: k => storage.delete(k) }, require(name) {
    if (name === 'react') return react;
    if (name === 'react/jsx-runtime') return require(root + '/node_modules/react/jsx-runtime');
    if (name === 'lucide-react') return require(root + '/node_modules/lucide-react');
    if (name.endsWith('financePresentation')) return load('features/accounting/financePresentation.ts');
    if (name.startsWith('../features/')) return { __esModule: true, default: () => null };
    if (name.endsWith('financeAccess')) return { fetchFinanceAccess: async () => ({ can_view_summary: true }) };
    if (name === '../services/api') return { fetchFinanceSummaryApi: async () => { if (failFinance) throw Error('报表连接失败'); return { year: '2026', months: [], annual: { output_value: '123.45', production_cost: '0', production_profit: '123.45', operating_profit: '123.45' } }; } };
    if (name.startsWith('../components/')) return { __esModule: true, default: () => null };
    if (name.endsWith('readRequestPool')) return { fetchReadRequest: async (url, options) => { sent = options; return response; } };
    if (name.endsWith('productionFlow')) return { flowDate: v => v?.slice(0, 10) || '', flowRequest: async (url, method) => { requests.push([url, method]); if (method) { saved++; return {}; } if (failPlan) throw Error('网络暂时异常'); return { can_manage: true, plan: { assignee: '已安排负责人', equipment: '原设备', estimated_hours: '2' } }; } };
    throw Error('Unexpected dependency: ' + name);
  } });
  return exports;
}
function nodes(t, predicate, result = []) { if (Array.isArray(t)) t.forEach(n => nodes(n, predicate, result)); else if (t?.props) { if (predicate(t)) result.push(t); nodes(t.props.children, predicate, result); } return result; }
function text(t) { return typeof t === 'string' || typeof t === 'number' ? String(t) : Array.isArray(t) ? t.map(text).join('') : t?.props ? text(t.props.children) : ''; }
async function run() {
  const Results = load('components/WorkbenchResultsState.tsx').default;
  let retried = 0, reset = 0;
  const props = { loading: false, error: '连接中断', empty: true, filtered: true, onRetry: () => retried++, onReset: () => reset++ };
  let tree = Results(props);
  assert.equal(tree.props.role, 'alert'); assert.ok(!text(tree).includes('没有找到'));
  nodes(tree, n => n.type === 'button')[0].props.onClick(); assert.equal(retried, 1);
  tree = Results({ ...props, error: '' }); nodes(tree, n => n.type === 'button')[0].props.onClick(); assert.equal(reset, 1);
  assert.equal(Results({ ...props, loading: true }).props.role, 'status');
  assert.equal(Results({ ...props, empty: false, error: '' }), null);
  const Plan = load('components/PartPlanningPanel.tsx').default;
  const planProps = { data: { flow: { item: { id: 1, quantity: 10, production_quantity: 12 } } }, onChanged: async () => {}, onBusyChange: () => {} };
  function render() { index = refIndex = effectIndex = 0; effects = []; return Plan(planProps); }
  render(); effects.forEach(f => f()); await new Promise(done => setImmediate(done)); tree = render();
  assert.ok(text(tree).includes('生产安排读取失败')); assert.equal(nodes(tree, n => n.type === 'form').length, 0);
  const retry = nodes(tree, n => n.type === 'button' && text(n) === '重新读取生产安排')[0]; assert.ok(retry); retry.props.onClick();
  failPlan = false; render(); effects.forEach(f => f()); await new Promise(done => setImmediate(done)); tree = render();
  assert.ok(nodes(tree, n => n.type === 'input' && n.props.value === '已安排负责人').length); assert.equal(saved, 0, 'Retry only reads; it must not erase an existing plan');
  state = []; deps = []; refs = [];
  const useQuery = load('hooks/useOrderQuery.ts').useOrderQuery;
  function query() { index = refIndex = effectIndex = 0; effects = []; return useQuery(); }
  let queries = query(); effects.forEach(f => f()); let calls = 0;
  await queries.run(async () => { calls++; if (calls === 1) throw Error('查询失败'); });
  queries = query(); assert.equal(queries.queryError, '查询失败'); assert.equal(queries.isSearching, false);
  queries.retry(); await new Promise(done => setImmediate(done)); queries = query(); assert.equal(calls, 2); assert.equal(queries.queryError, '');
  let rejectOld;
  const old = queries.run(() => new Promise((resolve, reject) => { rejectOld = reject; }));
  await queries.run(async () => {}); rejectOld(Error('过时失败')); await old;
  queries = query(); assert.equal(queries.queryError, '', 'Old failures must not overwrite the new result');
  state = []; deps = []; refs = []; failFinance = false;
  const Finance = load('pages/Finance.tsx').default;
  function finance() { index = refIndex = effectIndex = 0; effects = []; return Finance({}); }
  tree = finance(); assert.ok(text(tree).includes('读取中…')); assert.ok(!text(tree).includes('¥0.00'));
  effects.forEach(f => f()); await new Promise(done => setImmediate(done)); tree = finance(); assert.ok(text(tree).includes('123.45'));
  state = []; deps = []; refs = []; failFinance = true;
  finance(); effects.forEach(f => f()); await new Promise(done => setImmediate(done)); tree = finance();
  assert.ok(text(tree).includes('未读取')); assert.ok(!text(tree).includes('¥0.00')); assert.ok(!text(tree).includes('本年度暂无已确认'));
  const shared = load('components/shared.tsx');
  await shared.authFetch('/api/platform/test', { headers: new Headers({ 'Content-Type': 'application/json', 'X-Request-ID': 'ui-test' }) });
  assert.equal(sent.headers.get('Content-Type'), 'application/json'); assert.equal(sent.headers.get('X-Request-ID'), 'ui-test'); assert.equal(sent.headers.get('Authorization'), 'Bearer test-token');
  response = { status: 401, clone: () => ({ json: async () => ({ code: 4004 }) }) };
  await shared.authFetch('/api/platform/test'); assert.equal(storage.has('auth_token'), false); assert.equal(location.href, '/platform/');
  console.log('PASS: loading/error/empty feedback, retry/reset callbacks, failed plan reads hide editable forms, retry preserves saved plan, query failures handled with retry and stale-response protection, finance never presents unread values as zero, Headers forwarding, expired sessions return to ERP login.');
}
run().catch(e => { console.error(e); process.exitCode = 1; });
