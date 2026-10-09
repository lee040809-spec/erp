import assert from 'node:assert/strict';
import { test } from 'node:test';
import { saveProcessStatus } from './processStatus';

test('process saving reports network, server and business failures; returns server states only on success', async () => {
  const oldFetch = globalThis.fetch;
  const oldStorage = globalThis.localStorage;
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { getItem: () => null } });
  try {
    globalThis.fetch = async () => { throw new Error('network unavailable'); };
    await assert.rejects(saveProcessStatus(1, 1, 'completed'), /network unavailable/);
    globalThis.fetch = async () => new Response(JSON.stringify({error:'当前公司无该工序'}), {status:404});
    await assert.rejects(saveProcessStatus(1, 1, 'completed'), /当前公司无该工序/);
    globalThis.fetch = async () => new Response(JSON.stringify({success:false,message:'事务失败'}), {status:200});
    await assert.rejects(saveProcessStatus(1, 1, 'completed'), /事务失败/);
    globalThis.fetch = async () => new Response(JSON.stringify({success:true,item_status:'processing',order_status:'processing'}), {status:200});
    const result = await saveProcessStatus(1, 1, 'completed');
    assert.equal(result.item_status,'processing');
    assert.equal(result.order_status,'processing');
  } finally {
    globalThis.fetch = oldFetch;
    Object.defineProperty(globalThis, 'localStorage', {configurable:true,value:oldStorage});
  }
});
