const fs=require('fs'),path=require('path'),vm=require('vm'),assert=require('assert/strict');
const front=path.resolve(__dirname,'..'),ts=require(front+'/node_modules/typescript'),React=require(front+'/node_modules/react');
const actual={...React,useState:initial=>[typeof initial==='function'?initial():initial,()=>{}]};
const componentExports={};const js=ts.transpileModule(fs.readFileSync(front+'/src/components/ProcessCell.tsx','utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022,jsx:ts.JsxEmit.React,esModuleInterop:true}}).outputText;
vm.runInNewContext(js,{exports:componentExports,require:name=>name==='react'?actual:name==='lucide-react'?require(front+'/node_modules/lucide-react'):name==='./shared'?{PROCESS_COLORS:{}}:require(name)});
const Cell=componentExports.default;
let processes=[{id:1,name:'下料',status:'processing',actual_start_at:'2026-10-09',is_outsourced:false,outsourcing_fee:10},{id:2,name:'铣削',status:'pending',planned_start_at:'2026-10-10',planned_end_at:'2026-10-12',is_outsourced:false,outsourcing_fee:20}],updated;
function nodes(tree,predicate,result=[]){if(Array.isArray(tree)){tree.forEach(n=>nodes(n,predicate,result));return result}if(tree&&typeof tree==='object'&&tree.props){if(predicate(tree))result.push(tree);nodes(tree.props.children,predicate,result)}return result;}
const render=()=>Cell({processes,onUpdate:value=>updated=value,inline:true,routeOnly:true,hideCostFields:true});
let tree=render();nodes(tree,n=>n.type==='input'&&n.props['aria-label']==='工序1名称')[0].props.onChange({target:{value:'下料修订'}});assert.equal(updated[0].name,'下料修订');assert.equal(updated[0].id,1);assert.equal(updated[0].status,'processing');assert.equal(updated[0].actual_start_at,'2026-10-09');assert.equal(updated[0].outsourcing_fee,10);
nodes(tree,n=>n.type==='button'&&n.props['aria-label']==='下移工序1')[0].props.onClick();assert.equal(updated[1].id,1);assert.equal(updated[0].planned_end_at,'2026-10-12');
assert.equal(nodes(tree,n=>n.type==='button'&&n.props['aria-label']==='删除工序1')[0].props.disabled,true);
assert.equal(nodes(tree,n=>n.type==='select'&&n.props['aria-label']==='工序1生产状态')[0].props.disabled,true);
nodes(tree,n=>n.type==='button'&&n.props['aria-label']==='删除工序2')[0].props.onClick();assert.equal(updated.length,1);assert.equal(updated[0].id,1);
const add=nodes(tree,n=>n.type==='button'&&Array.isArray(n.props.children)&&n.props.children.includes('检验'));
const cut=nodes(tree,n=>n.type==='button'&&Array.isArray(n.props.children)&&n.props.children.includes('下料'))[0];assert.ok(cut);cut.props.onClick();assert.equal(updated.length,3);assert.equal(updated[2].status,'pending');assert.equal(updated[2].id,undefined);
console.log('PASS: inline route editor rename/reorder/add/delete events retain IDs, actual dates, plans and fees; production state and used-step deletion are protected. Browser layout was not exercised.');