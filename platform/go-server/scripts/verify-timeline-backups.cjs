// Compare SQL literal rows by original columns; quoted commas and escaped quotes are preserved.
const fs = require('node:fs');
const crypto = require('node:crypto');
const path = require('node:path');
const root = path.resolve(__dirname, '../../..');
const beforePath = path.join(root, '数据库备份/platform_test-before-process-timeline-20261006.sql');
const afterPath = path.join(root, '数据库备份/platform_test-after-process-timeline-20261006.sql');
function parse(sql) {
  const tables = new Map([...sql.matchAll(/CREATE TABLE `([^`]+)`/g)].map(m => [m[1], { columns: [], rows: [] }]));
  const headers = /INSERT INTO `([^`]+)` \(([^\n]+)\) VALUES\r?\n/g;
  let match;
  while ((match = headers.exec(sql))) {
    const table = tables.get(match[1]);
    table.columns = [...match[2].matchAll(/`([^`]+)`/g)].map(m => m[1]);
    let quoted = false, escaped = false, start = -1, fieldStart = -1, fields = [];
    for (let i = headers.lastIndex; i < sql.length; i++) {
      const ch = sql[i];
      if (quoted) {
        if (escaped) escaped = false;
        else if (ch === '\\') escaped = true;
        else if (ch === "'") quoted = false;
        continue;
      }
      if (ch === "'") { quoted = true; continue; }
      if (ch === ';') { headers.lastIndex = i + 1; break; }
      if (ch === '(') { start = i; fieldStart = i + 1; fields = []; }
      if (start >= 0 && (ch === ',' || ch === ')')) {
        fields.push(sql.slice(fieldStart, i).trim()); fieldStart = i + 1;
        if (ch === ')') {
          if (fields.length !== table.columns.length) throw new Error('Invalid SQL row: ' + match[1]);
          table.rows.push(fields); start = -1;
        }
      }
    }
  }
  return tables;
}
const beforeSQL = fs.readFileSync(beforePath, 'utf8'), afterSQL = fs.readFileSync(afterPath, 'utf8');
const before = parse(beforeSQL), after = parse(afterSQL), hash = v => crypto.createHash('sha256').update(v).digest('hex');
const checks = [];
for (const [name, old] of before) {
  const next = after.get(name);
  if (!next) throw new Error('Missing original table: ' + name);
  const indexes = old.columns.map(c => next.columns.indexOf(c));
  if (indexes.some(i => i < 0)) throw new Error('Missing original column: ' + name);
  const canonical = rows => rows.map(r => JSON.stringify(r)).sort().join('\n');
  const previousHash = hash(canonical(old.rows)), nextHash = hash(canonical(next.rows.map(r => indexes.map(i => r[i]))));
  const unchanged = old.rows.length === next.rows.length && previousHash === nextHash;
  checks.push({ table: name, before_rows: old.rows.length, after_rows: next.rows.length, original_columns: old.columns.length, original_data_unchanged: unchanged, before_sha256: previousHash, after_sha256: nextHash });
}
const processes = after.get('order_processes');
const nodes = ['planned_start_at', 'planned_end_at', 'actual_start_at', 'actual_end_at'].map(c => processes.columns.indexOf(c));
if (nodes.some(i => i < 0)) throw new Error('Missing process time column');
const timedRows = processes.rows.filter(r => nodes.some(i => r[i] !== 'NULL')).length;
const result = { checked_at: new Date().toISOString(), before_backup: beforePath, after_backup: afterPath, before_backup_sha256: hash(beforeSQL), after_backup_sha256: hash(afterSQL), original_data_unchanged: checks.every(c => c.original_data_unchanged), original_table_count: checks.length, historical_timed_rows: timedRows, new_event_rows: after.get('process_time_events')?.rows.length, tables: checks };
fs.writeFileSync(path.join(root, '方案/工序时间节点迁移前后核对-20261006.json'), JSON.stringify(result, null, 2));
if (!result.original_data_unchanged || timedRows !== 0 || result.new_event_rows !== 0) throw new Error('Migration data verification failed');
console.log(`Verified ${checks.length} original tables: all original literal rows unchanged; ${processes.rows.length} historical processes retain NULL time nodes.`);
