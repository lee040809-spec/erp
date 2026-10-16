const fs=require('fs'),path=require('path'),vm=require('vm'),assert=require('assert/strict');
const root=path.resolve(__dirname,'..'),ts=require(root+'/node_modules/typescript');
const exportsObject={};
vm.runInNewContext(ts.transpileModule(fs.readFileSync(root+'/src/utils/dialogEscape.ts','utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText,{exports:exportsObject});
let scopes=[],listener,fullscreen=null,closed=[],fullscreenExits=0;
const doc={get fullscreenElement(){return fullscreen},exitFullscreen(){fullscreenExits++;fullscreen=null;return Promise.resolve()},defaultView:{getComputedStyle:e=>({zIndex:e.z??'auto',visibility:e.hidden?'hidden':'visible'})},querySelectorAll:()=>scopes,addEventListener:(name,fn,capture)=>{assert.equal(capture,true);listener=fn},removeEventListener:(name,fn,capture)=>{assert.equal(fn,listener);assert.equal(capture,true);listener=null}};
function dialog(name,z,parent=null){const scope={name,z,parentElement:parent,hidden:false,blocked:false,buttons:[],getClientRects(){return this.hidden?[]:[{}]},getAttribute(){return this.blocked?'true':null},querySelectorAll(){return this.buttons}};const button={hidden:false,disabled:false,getClientRects(){return this.hidden?[]:[{}]},closest:()=>scope,matches(){return this.disabled},click(){closed.push(name)}};scope.buttons=[button];return scope}
function key(extra={}){const event={key:'Escape',repeat:false,isComposing:false,prevented:false,stopped:false,preventDefault(){this.prevented=true},stopImmediatePropagation(){this.stopped=true},...extra};listener(event);return event}
const cleanup=exportsObject.installDialogEscape(doc);
assert.equal(key().prevented,false);
const parent=dialog('order',100),child=dialog('drawing',200);scopes=[parent,child];key();assert.deepEqual(closed,['drawing']);
child.blocked=true;key();assert.equal(closed.length,1,'Busy child cannot close underlying order');child.blocked=false;child.buttons[0].disabled=true;key();assert.equal(closed.length,1);child.buttons[0].disabled=false;
child.hidden=true;key();assert.equal(closed.at(-1),'order');child.hidden=false;
closed=[];scopes=[parent,dialog('same-layer-later',100)];key();assert.deepEqual(closed,['same-layer-later']);
closed=[];const lower=dialog('lower',50),nested=dialog('nested',1000,lower),upper=dialog('upper',100);scopes=[lower,nested,upper];key();assert.deepEqual(closed,['upper'],'Respect ancestor stacking contexts');
closed=[];fullscreen={};const full=key();assert.equal(full.prevented,true);assert.equal(full.stopped,true);assert.equal(closed.length,0);assert.equal(fullscreenExits,1);
key({repeat:true});key({isComposing:true});key({key:'Enter'});assert.equal(closed.length,0);
const guarded=dialog('guarded',200);guarded.buttons[0].click=()=>{};scopes=[parent,guarded];key();assert.equal(closed.length,0,'Rejected close must not fall through to parent');
cleanup();assert.equal(listener,null);
// Every marked overlay must have an explicitly marked cancel action of its own.
function files(dir){return fs.readdirSync(dir,{withFileTypes:true}).flatMap(e=>e.isDirectory()?files(path.join(dir,e.name)):e.name.endsWith('.tsx')?[path.join(dir,e.name)]:[])}
let coverage=0;
for(const file of files(root+'/src')){const source=fs.readFileSync(file,'utf8'),ast=ts.createSourceFile(file,source,ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
 const has=(node,name)=>node.attributes.properties.some(a=>ts.isJsxAttribute(a)&&a.name.getText(ast)===name);
 function visit(node){if(ts.isJsxElement(node)&&has(node.openingElement,'data-escape-dialog')){coverage++;let cancels=0;function children(child){if(child!==node&&ts.isJsxElement(child)&&has(child.openingElement,'data-escape-dialog'))return;if(ts.isJsxElement(child)&&child.openingElement.tagName.getText(ast)==='button'&&has(child.openingElement,'data-escape-close'))cancels++;ts.forEachChild(child,children)}children(node);assert.ok(cancels,`Missing cancel action in ${file}:${ast.getLineAndCharacterOfPosition(node.pos).line+1}`)}ts.forEachChild(node,visit)}visit(ast)
}
assert.ok(coverage>=35);console.log(`PASS: ${coverage} dialog scopes, top-layer cancellation, nested stacking contexts, busy/disabled protection, fullscreen isolation, held Escape/IME, close rejection and listener cleanup. No business data writes.`);
