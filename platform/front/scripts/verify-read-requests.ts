import assert from 'node:assert/strict';
import {createReadRequestPool} from '../src/services/readRequestPool';
async function main(){
 let calls=0;let resolve!: (response:Response)=>void;
 const pool=createReadRequestPool((async()=>{calls++;return new Promise<Response>(r=>{resolve=r});}) as typeof fetch);
 const a=pool('/api/platform/orders'),b=pool('/api/platform/orders');assert.equal(calls,1);
 resolve(new Response('{"data":[1]}'));assert.deepEqual(await(await a).json(),await(await b).json());
 const c=pool('/api/platform/orders');assert.equal(calls,2);resolve(new Response('{}'));await c;
 let isolatedCalls=0;
 const isolated=createReadRequestPool((async()=>{isolatedCalls++;return new Response('{}');}) as typeof fetch);
 await Promise.all([isolated('/api/platform/orders',{headers:{Authorization:'Bearer A'}}),isolated('/api/platform/orders',{headers:{Authorization:'Bearer B'}})]);
 assert.equal(isolatedCalls,2);
 await Promise.all([isolated('/api/platform/orders',{method:'POST'}),isolated('/api/platform/orders',{method:'POST'})]);assert.equal(isolatedCalls,4);
 let retries=0;const failed=createReadRequestPool((async()=>{retries++;throw new Error('offline')}) as typeof fetch);
 await Promise.allSettled([failed('/api/platform/orders'),failed('/api/platform/orders')]);assert.equal(retries,1);
 await failed('/api/platform/orders').catch(()=>{});assert.equal(retries,2);
 const resolvers:((r:Response)=>void)[]=[];
 const changed=createReadRequestPool((async()=>new Promise<Response>(r=>resolvers.push(r))) as typeof fetch);
 const old=changed('/api/platform/orders');const write=changed('/api/platform/orders',{method:'PATCH'});
 const fresh=changed('/api/platform/orders');assert.equal(resolvers.length,3);
 resolvers[0](new Response('old'));assert.equal(await(await old).text(),'old');
 const shared=changed('/api/platform/orders');assert.equal(resolvers.length,3);
 resolvers[2](new Response('fresh'));assert.equal(await(await fresh).text(),'fresh');assert.equal(await(await shared).text(),'fresh');
 resolvers[1](new Response('{}'));await write;
 const after=changed('/api/platform/orders');assert.equal(resolvers.length,4);resolvers[3](new Response('after'));assert.equal(await(await after).text(),'after');
 console.log('PASS simultaneous reads, response isolation, fresh reloads, account isolation, mutation invalidation, failure retry');
}
main().catch(e=>{console.error(e);process.exitCode=1});
