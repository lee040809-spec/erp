const fs=require('fs'),path=require('path'),vm=require('vm');
const root=path.resolve(__dirname,'../..'),ts=require(root+'/node_modules/typescript');
function harness(){
 let current,nextTimer=0;const timers=new Map(),store=new Map(),modules=new Map(),listeners=new Map();
 const storage={getItem:k=>store.get(k)??null,setItem:(k,v)=>store.set(k,String(v)),removeItem:k=>store.delete(k)};
 const equal=(a,b)=>a&&b&&a.length===b.length&&a.every((v,i)=>Object.is(v,b[i]));
 const react={useCallback:fn=>fn,useState(value){const c=current,i=c.s++;if(!(i in c.states))c.states[i]=typeof value==='function'?value():value;return[c.states[i],v=>c.states[i]=typeof v==='function'?v(c.states[i]):v]},useRef(value){const c=current,i=c.r++;return c.refs[i]||(c.refs[i]={current:value})},useEffect(fn,deps){const c=current,i=c.e++;if(!equal(c.deps[i],deps)){c.deps[i]=deps;c.pending.push(()=>{c.cleanups[i]?.();c.cleanups[i]=fn()})}}};
 const window={addEventListener:(name,fn)=>listeners.set(name,fn),removeEventListener:(name,fn)=>{if(listeners.get(name)===fn)listeners.delete(name)}};
 function load(file){if(modules.has(file))return modules.get(file);const exports={};
  const output=ts.transpileModule(fs.readFileSync(root+'/src/'+file,'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022,jsx:ts.JsxEmit.ReactJSX}}).outputText;
  vm.runInNewContext(output,{exports,Date,JSON,Number,Math,localStorage:storage,window,setTimeout:fn=>{const id=++nextTimer;timers.set(id,fn);return id},clearTimeout:id=>timers.delete(id),require(name){if(name==='react')return react;if(name==='react/jsx-runtime')return require(root+'/node_modules/react/jsx-runtime');const target=path.posix.normalize(path.posix.join(path.posix.dirname(file),name));for(const ext of ['.ts','.tsx'])if(fs.existsSync(root+'/src/'+target+ext))return load(target+ext);throw Error('unexpected '+name)}});modules.set(file,exports);return exports;
 }
 function runner(fn){const c={states:[],refs:[],deps:[],cleanups:[],pending:[]};return{render(){current=c;c.s=c.r=c.e=0;c.pending=[];const value=fn();c.pending.forEach(f=>f());return value},close(){c.cleanups.forEach(f=>f?.())}}}
 const tick=()=>{const pending=[...timers.values()];timers.clear();pending.forEach(f=>f())};
 return {load,runner,tick,store,storage,listeners};
}
module.exports={harness};
