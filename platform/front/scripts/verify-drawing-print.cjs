const fs=require('fs'),path=require('path'),vm=require('vm'),assert=require('assert/strict');
const root=path.resolve(__dirname,'..'),ts=require(root+'/node_modules/typescript');
let frames=[],created=[],revoked=[],printed=[],rendered=[],cancelled=0,failRender=false,silent=false,full=false;
const body={appendChild(frame){frame.host='body';frames.push(frame);}},fullscreen={appendChild(frame){frame.host='fullscreen';frames.push(frame);}};
function element(tag){const item={tag,children:[],style:{},appendChild(child){this.children.push(child);},setAttribute(){},remove(){this.removed=true;}};
 if(tag==='img'){item.naturalWidth=1200;item.naturalHeight=800;Object.defineProperty(item,'src',{set(value){this.source=value;queueMicrotask(()=>this.onload?.());}});}
 if(tag==='canvas'){item.getContext=()=>({});item.toBlob=fn=>fn({type:'image/png'});}
 return item;
}
function frame(){const iframe=element('iframe'),events=new EventTarget();const doc={head:element('head'),body:element('body'),createElement:element,open(){},close(){},write(html){this.html=html;}};
 iframe.contentDocument=doc;iframe.contentWindow={addEventListener:(...args)=>events.addEventListener(...args),removeEventListener:(...args)=>events.removeEventListener(...args),focus(){},print(){printed.push({iframe,pages:doc.body.children.slice()});if(!silent)queueMicrotask(()=>events.dispatchEvent(new Event('afterprint')));}};return iframe;
}
const output={};vm.runInNewContext(ts.transpileModule(fs.readFileSync(root+'/src/features/drawings/printDrawing.ts','utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText,{exports:output,DOMException,Error,Math,Promise,URL:{createObjectURL(){const url='blob:print-'+created.length;created.push(url);return url;},revokeObjectURL(url){revoked.push(url);}},setTimeout,clearTimeout,document:{body,get fullscreenElement(){return full?fullscreen:null;},createElement:tag=>tag==='iframe'?frame():element(tag)}});
const pdf={numPages:2,async getPage(number){return{rotate:number===1?0:90,getViewport({scale,rotation}){return{width:(rotation%180?842:595)*scale,height:(rotation%180?595:842)*scale,rotation};},render({canvas,viewport}){rendered.push({number,viewport,pixels:canvas.width*canvas.height,canvas});return{promise:failRender?Promise.reject(Error('fixture render failed')):Promise.resolve(),cancel(){cancelled++;}};}};}};
const tick=()=>new Promise(done=>setImmediate(done));
async function run(){
 await output.printDrawing({pdf,title:'<script>fixture</script>.pdf',rotation:0,signal:new AbortController().signal});
 assert.equal(printed.length,1);assert.equal(printed[0].pages.length,2);assert.deepEqual(rendered.map(r=>r.number),[1,2]);assert.ok(rendered.every(r=>r.pixels<16100000));assert.ok(rendered.every(r=>r.canvas.width===0&&r.canvas.height===0));
 assert.equal(printed[0].iframe.contentDocument.title,'<script>fixture</script>.pdf');assert.ok(!printed[0].iframe.contentDocument.html.includes('fixture'));assert.ok(printed[0].iframe.contentDocument.head.children[0].textContent.includes('size:842pt 595pt'));
 assert.ok(frames.at(-1).removed);assert.deepEqual(revoked,created);
 full=true;await output.printDrawing({imageUrl:'blob:preview-image',title:'image.png',rotation:90,signal:new AbortController().signal});assert.equal(frames.at(-1).host,'fullscreen');assert.equal(printed.at(-1).pages.length,1);assert.ok(printed.at(-1).pages[0].children[0].style.cssText.includes('rotate(90deg)'));assert.equal(revoked.includes('blob:preview-image'),false,'Do not revoke the preview-owned image');
 full=false;silent=true;const control=new AbortController();const pending=output.printDrawing({imageUrl:'blob:preview-image',title:'cancel',signal:control.signal});await tick();control.abort();await assert.rejects(pending,{name:'AbortError'});assert.ok(frames.at(-1).removed);
 silent=false;failRender=true;const count=printed.length;await assert.rejects(output.printDrawing({pdf,title:'fail',signal:new AbortController().signal}),/fixture render failed/);assert.equal(printed.length,count);assert.ok(frames.at(-1).removed);
 const early=new AbortController();early.abort();const previous=frames.length;await assert.rejects(output.printDrawing({pdf,title:'early',signal:early.signal}),{name:'AbortError'});assert.equal(frames.length,previous);
 console.log('PASS: complete two-page PDF printing, per-page paper sizes, bounded canvas, released page resources, image rotation, fullscreen host, cancelled/failed preparation cleanup and no preview URL revocation. Browser printing mocked; no print job or business writes.');
}
run().catch(error=>{console.error(error);process.exitCode=1;});
