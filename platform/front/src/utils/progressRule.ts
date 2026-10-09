// Shared safe parser for the exact progress range supported by the server.
export function parseProgressRule(formula: string): { left: number; right: number; descending: boolean } | null {
 const clean = formula.replace(/\s/g, '').replaceAll('{交货日期}', '{订单交期}');
 const match = clean.match(/^(-?(?:\d+(?:\.\d+)?|\.\d+)(?:\/\d+(?:\.\d+)?)?)(<=|>=)(.+)(<=|>=)(-?(?:\d+(?:\.\d+)?|\.\d+)(?:\/\d+(?:\.\d+)?)?)$/);
 if (!match || match[2] !== match[4]) return null;
 let expression=match[3];
 while(expression.startsWith('(')){let depth=0,end=-1;for(let i=0;i<expression.length;i++){if(expression[i]==='(')depth++;if(expression[i]===')')depth--;if(depth===0){end=i;break;}}if(end!==expression.length-1)break;expression=expression.slice(1,-1);}
 if(expression !== '({当天}-{订单日期})/({订单交期}-{订单日期})')return null;
 const numeric=(value:string)=>{const [a,b]=value.split('/').map(Number);if(!Number.isFinite(a)|| (b!==undefined&&(!Number.isFinite(b)||b===0)))return NaN;return b===undefined?a:a/b;};
 const left=numeric(match[1]),right=numeric(match[5]);if(!Number.isFinite(left)||!Number.isFinite(right))return null;
 return {left,right,descending:match[2]==='>='};
}
export function progressRuleMatches(rule:NonNullable<ReturnType<typeof parseProgressRule>>,progress:number):boolean {
 return Number.isFinite(progress) && (rule.descending ? rule.left>=progress && progress>=rule.right : rule.left<=progress && progress<=rule.right);
}
