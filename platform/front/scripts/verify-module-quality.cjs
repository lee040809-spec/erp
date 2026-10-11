const fs=require('fs'),path=require('path'),vm=require('vm'),assert=require('assert/strict'),crypto=require('crypto');
const front=path.resolve(__dirname,'..'),ts=require(front+'/node_modules/typescript'),React=require(front+'/node_modules/react');
const modules=new Map();let effects=[],timers=new Map(),nextTimer=0,states=[],refs=[],stateIndex=0,refIndex=0,parentBusy=[],server;
function Editor(){return null}function Stub(){return null}
const react={...React,useMemo:f=>f(),useCallback:f=>f,useEffect:f=>effects.push(f),useState(initial){const i=stateIndex++;if(!(i in states))states[i]=typeof initial==='function'?initial():initial;return[states[i],v=>states[i]=typeof v==='function'?v(states[i]):v]},useRef(initial){const i=refIndex++;if(!(i in refs))refs[i]={current:initial};return refs[i]}};
function load(file){if(modules.has(file))return modules.get(file);const exports={};
const code=ts.transpileModule(fs.readFileSync(front+'/src/'+file,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
vm.runInNewContext(code,{exports,console,Date,Intl,Math,Uint8Array,atob,btoa,setTimeout:f=>{const id=++nextTimer;timers.set(id,f);return id},clearTimeout:id=>timers.delete(id),require(name){if(name==='react')return react;if(name==='react/jsx-runtime')return require(front+'/node_modules/react/jsx-runtime');if(name==='pako')return require(front+'/node_modules/pako');if(name==='lucide-react')return require(front+'/node_modules/lucide-react');if(name.endsWith('/templateCodec'))return load('utils/templateCodec.ts');if(name.endsWith('/productionFlow'))return {flowDate:v=>v?.slice(0,10)||'',flowToday:()=> '2026-10-11',flowRequest:async()=>server,withRequestID:(_,payload)=>payload};if(name.endsWith('/OutsourcingDispatchEditor'))return {__esModule:true,default:Editor};if(name.startsWith('./'))return {__esModule:true,default:Stub};throw Error('Unexpected dependency '+name)}});modules.set(file,exports);return exports;}
function nodes(t,p,result=[]){if(Array.isArray(t))t.forEach(n=>nodes(n,p,result));else if(t?.props){if(p(t))result.push(t);nodes(t.props.children,p,result)}return result}
function text(t){if(typeof t==='string'||typeof t==='number')return String(t);if(Array.isArray(t))return t.map(text).join('');return t?.props?text(t.props.children):''}
async function run(){
 const replace=load('utils/templateReplace.ts'),data={order_number:'$&-ORDER',unit_price:'3.2',total_amount:'1000.00',other:{small_total:'1000.00',big_total:'壹仟元整',total_quantity:10,delivery_number:'DEL-1',reconciliation_number:'REC-1'}};
 const content='{{order.order_number}} / {{other.small_total}} / {{other.big_total}} / {{other.total_quantity}} / {{other.small_total}}';
 const expected='$&-ORDER / 1000.00 / 壹仟元整 / 10 / 1000.00';
 assert.equal(replace.replaceDeliveryOrderPlaceholders(content,data),expected);
 assert.equal(replace.replaceReconciliationOrderPlaceholders(content,data),expected);
 assert.equal(replace.getFieldValue(data,'unit_price'),'3.20');
 assert.equal(replace.getFieldValue({unit_price:0},'unit_price'),'0');
 assert.equal(replace.getFieldValue({unit_price:null},'unit_price'),'');
 assert.equal(replace.replaceItemPlaceholders('{{#items.part_name}} {{#items.part_name}}',{part_name:'$&零件'}),'$&零件 $&零件');
 assert.equal(replace.replaceDeliveryOrderPlaceholders('{{other.delivery_number}}',data),'DEL-1');
 assert.equal(replace.replaceReconciliationOrderPlaceholders('{{other.reconciliation_number}}',data),'REC-1');
 const codec=load('utils/templateCodec.ts'),workbook={sheetOrder:['s1'],sheets:{s1:{name:'送货单',cellData:{}}},random:crypto.randomBytes(200000).toString('base64')},encoded=codec.encodePrintTemplate(workbook);
 assert.ok(encoded.length>200000,'Exercise compression larger than JS spread argument limits');
 for(const input of [encoded,'GZIP:'+encoded.slice(7),JSON.stringify(workbook)])assert.equal(JSON.stringify(codec.decodePrintTemplate(input)),JSON.stringify(workbook));
 assert.throws(()=>codec.decodePrintTemplate('UNIVER:%%%'));
 const preview=load('hooks/usePrintPreview.ts');
 assert.ok(preview.usePrintWorkbook('broken').error);assert.equal(preview.usePrintWorkbook(encoded).workbook.sheets.s1.name,'送货单');
 let filled=0,ready=false;const editor={current:{isReady:()=>ready}};effects=[];
 preview.usePrintPreview('preview',data,workbook,editor,()=>filled++);
 const cleanup=effects.pop()();assert.equal(timers.size,1);const pending=[...timers.values()][0];cleanup();assert.equal(timers.size,0);ready=true;pending();assert.equal(filled,0,'Closed preview must never fill');
 effects=[];ready=false;preview.usePrintPreview('preview',data,workbook,editor,()=>filled++);const cleanup2=effects.pop()();ready=true;const [id,callback]=[...timers.entries()][0];timers.delete(id);callback();assert.equal(filled,1);assert.equal(timers.size,0);cleanup2();
 effects=[];preview.usePrintPreview('config',data,workbook,editor,()=>filled++);assert.equal(effects.pop()(),undefined);assert.equal(timers.size,0);
 const FieldButtons=load('components/PrintTemplateFieldButtons.tsx').default;let selected;
 const field={key:'items.part_name',label:'零件名称',isList:true};const button=nodes(FieldButtons({fields:[field],tone:'item',onSelect:v=>selected=v}),n=>n.type==='button')[0];assert.ok(text(button).includes('{{#items.part_name}}'));button.props.onClick();assert.equal(selected,field);
 // Run the actual parent callback wiring, without writing any ERP data.
 states=[];refs=[];effects=[];const Panel=load('components/OutsourcingPanel.tsx').default;
 const flow={flow:{item:{id:1},processes:[{id:1,is_outsourced:true,name:'铣削'}],batches:[],tracking:true},can_manage:true,can_settle:false};
 server={items:[{id:1,supplier_name:'测试供应商',order_process_id:1,status:'dispatched',quantity:10,open_quantity:10}],settlements:[],events:[]};
 const props={data:flow,onChanged:async()=>{},onBusyChange:v=>parentBusy.push(v)};
 function render(){stateIndex=refIndex=0;effects=[];return Panel(props)}
 let tree=render();effects.forEach(f=>f());await new Promise(done=>setImmediate(done));tree=render();
 nodes(tree,n=>n.type==='button'&&text(n)==='修改发出记录')[0].props.onClick();tree=render();const editorNode=nodes(tree,n=>n.type===Editor)[0];assert.equal(typeof editorNode.props.onBusyChange,'function');
 editorNode.props.onBusyChange(true);render();effects.forEach(f=>f());assert.equal(parentBusy.at(-1),true,'Editing dispatch must block workspace close');
 editorNode.props.onBusyChange(false);render();effects.forEach(f=>f());assert.equal(parentBusy.at(-1),false);
 console.log('PASS: repeated/mixed placeholders, literal dollar values, DECIMAL strings, all template formats, large workbook round trip, cancelled preview timer, one-time ready fill, field buttons and outsourcing editor parent save guard.');
}run().catch(e=>{console.error(e);process.exitCode=1});
