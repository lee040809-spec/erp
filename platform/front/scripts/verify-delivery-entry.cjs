const fs=require('fs'),path=require('path'),vm=require('vm'),assert=require('assert/strict');
const root=path.resolve(__dirname,'..'),ts=require(root+'/node_modules/typescript'),React=require(root+'/node_modules/react');
let states=[],refs=[],hook=0,ref=0,effects=[],requests=[],serverData,flowData,changed=0,priceEdits=0;
const react={...React,useState(initial){const i=hook++;if(!(i in states))states[i]=typeof initial==='function'?initial():initial;return [states[i],v=>states[i]=typeof v==='function'?v(states[i]):v];},useRef(initial){const i=ref++;if(!(i in refs))refs[i]={current:initial};return refs[i]},useEffect(f){effects.push(f)}};
function Empty(){return null}
let Modal,Completion,entry;
const policy={operation_notes:false,review_notes:false};
function load(relative){const exports={};const code=ts.transpileModule(fs.readFileSync(root+'/src/'+relative,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
vm.runInNewContext(code,{exports,Date,Intl,Math,console,crypto:{randomUUID:()=> 'frontend-request-1'},require(name){
if(name==='react')return react;if(name==='react/jsx-runtime')return require(root+'/node_modules/react/jsx-runtime');if(name==='react/jsx-runtime')return require(root+'/node_modules/react/jsx-runtime');
if(name==='lucide-react')return require(root+'/node_modules/lucide-react');
if(name.includes('businessDate'))return {businessToday:()=> '2026-10-10'};
if(name.includes('deliveryEntry'))return entry;
if(name.endsWith('/shared'))return {formatDate:x=>x,authFetch:async(url,options)=>{requests.push({url,options});return {ok:true,json:async()=>({data:options?{}:serverData})}}};
if(name.includes('productionFlow'))return {flowRequest:async()=>flowData};
if(name.includes('services/api'))return {fetchCompletionsApi:async()=>({remaining_quantity:1}),createCompletionApi:async(id,payload)=>{requests.push({id,payload})}};
if(name.includes('BusinessEntrySettings'))return {basicEvidence:policy,readEntryPolicy:async()=>({effective:policy})};
if(name==='./CompletionModal')return {__esModule:true,default:Completion};
return {__esModule:true,default:Empty};
}});return exports;}
entry=load('utils/deliveryEntry.ts');Completion=load('components/CompletionModal.tsx').default;Modal=load('components/OrderBusinessModal.tsx').default;
function nodes(tree,predicate,result=[]){if(Array.isArray(tree)){tree.forEach(n=>nodes(n,predicate,result));return result;}if(tree&&tree.props){if(predicate(tree))result.push(tree);nodes(tree.props.children,predicate,result)}return result}
function text(tree){if(typeof tree==='string'||typeof tree==='number')return String(tree);if(Array.isArray(tree))return tree.map(text).join('');return tree?.props?text(tree.props.children):''}
const baseItem={id:523,part_name:'探测器后盖',quantity:1,production_quantity:1,scrap_quantity:0,qualified_quantity:0,delivered_quantity:0,available_delivery:0,price_ready:0,production_ready:1};
const props={order:{id:142,order_number:'YHS-20261007-001',items:[{id:523,unit_price:0}]},initialKind:'delivery',initialItem:523,onClose(){},onChanged(){changed++},onEditOrder(){priceEdits++}};
function reset(item=baseItem){states=[];refs=[];states[0]='delivery';states[1]={items:[item],records:[],completions:[],can_complete:true,can_delivery:true,evidence:policy};states[2]='523';states[3]='1';states[4]='2026-10-09';states[13]=false;states[18]='';states[19]={flow:{item:{id:523},tracking:false,batches:[],available_completion:0}};serverData=states[1];flowData=states[19];requests=[];}
function render(component=Modal,p=props){hook=0;ref=0;effects=[];return component(p)}
function button(tree,label){const found=nodes(tree,n=>n.type==='button'&&text(n)===label)[0];assert.ok(found,'Missing button '+label);return found}
async function run(){
reset();let tree=render();assert.ok(text(tree).includes('尚未录入有效零件单价'));assert.equal(button(tree,'先确认合格完工').props.disabled,true);assert.equal(button(tree,'保存本次登记').props.disabled,true);
button(tree,'补录零件单价').props.onClick();assert.equal(priceEdits,1);
const quantity=nodes(tree,n=>n.type==='input'&&n.props.type==='number')[0];assert.equal(quantity.props.max,undefined);assert.equal(quantity.props['aria-describedby'],'delivery-quantity-guidance');
await nodes(tree,n=>n.type==='form')[0].props.onSubmit({preventDefault(){}});assert.equal(requests.length,0);assert.ok(states[10].includes('确认合格完工'));
reset({...baseItem,price_ready:1});tree=render();assert.equal(button(tree,'先确认合格完工').props.disabled,false);button(tree,'先确认合格完工').props.onClick();
tree=render();let modal=nodes(tree,n=>n.type===Completion)[0];assert.equal(modal.props.suggestedQuantity,1);assert.equal(modal.props.latestBusinessDate,'2026-10-09');
serverData={...serverData,items:[{...baseItem,price_ready:1,qualified_quantity:1,available_delivery:1}]};await modal.props.onSuccess();
assert.equal(states[0],'delivery');assert.equal(states[3],'1');assert.equal(changed,1);
tree=render();assert.equal(button(tree,'保存本次登记').props.disabled,false);assert.equal(nodes(tree,n=>n.type==='input'&&n.props.type==='number')[0].props.max,1);
await nodes(tree,n=>n.type==='form')[0].props.onSubmit({preventDefault(){}});const posted=requests.find(r=>r.options?.method==='POST');assert.ok(posted);const body=JSON.parse(posted.options.body);assert.equal(body.quantity,1);assert.equal(body.business_date,'2026-10-09');assert.equal(body.kind,'delivery');assert.equal(states[0],'records');
const done=entry.deliveryEntryState({...baseItem,qualified_quantity:1,delivered_quantity:1});assert.equal(done.limit,0);assert.equal(done.needsCompletion,false);assert.ok(done.message.includes('无需重复登记'));
assert.equal(entry.deliveryEntryState({...baseItem,quantity:10,production_quantity:12,qualified_quantity:12,available_delivery:10}).limit,10);
const eligible={...baseItem,quantity:10,production_quantity:10,qualified_quantity:10,available_delivery:10};
const batch={quantity:5,production_quantity:5,qualified_quantity:2,delivered_quantity:1};
assert.equal(entry.deliveryEntryState(eligible,undefined,true).limit,0);assert.equal(entry.deliveryEntryState(eligible,batch,true).limit,1);
reset({...eligible,id:523,price_ready:1});states[18]='8';states[19]={flow:{item:{id:523},tracking:true,available_completion:3,batches:[{id:8,...batch}]}};
tree=render();assert.equal(nodes(tree,n=>n.type==='input'&&n.props.type==='number')[0].props.max,1);
states[3]='2';tree=render();await nodes(tree,n=>n.type==='form')[0].props.onSubmit({preventDefault(){}});assert.ok(states[10].includes('最多可交 1'));assert.equal(requests.length,0);
states=[];refs=[];requests=[];const cp={item:{item_id:523,part_name:'探测器后盖',quantity:1},suggestedQuantity:1,latestBusinessDate:'2026-10-09',onClose(){},onSuccess(){}};
render(Completion,cp);await Promise.all(effects.map(f=>f()));await new Promise(r=>setImmediate(r));tree=render(Completion,cp);assert.equal(nodes(tree,n=>n.type==='input'&&n.props.type==='date')[0].props.max,'2026-10-09');assert.equal(states[3],'2026-10-09');
states[3]='2026-10-10';tree=render(Completion,cp);await nodes(tree,n=>n.type==='form')[0].props.onSubmit({preventDefault(){}});assert.equal(requests.length,0);assert.ok(states[7].includes('不能晚于交货'));
console.log('PASS: missing-price/completion guidance, direct price edit, guarded delivery submission, completion return preserving quantity/date, refreshed delivery max, batch cap, duplicate/spare guards and completion-before-delivery date. Actual component callbacks exercised; browser layout not exercised.');
}
run().catch(e=>{console.error(e);process.exitCode=1});
