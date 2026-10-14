import {t,locale} from "./i18n.js";
export const $=id=>document.getElementById(id);
export function node(tag,text,className){
  const el=document.createElement(tag);
  if(text!==undefined) el.textContent=String(text);
  if(className) el.className=className;
  return el;
}
export function button(key,handler,danger=false){
  const el=node("button",t(key),danger ? "danger":"");
  el.type="button";
  el.dataset.i18n=key;
  el.addEventListener("click",()=>Promise.resolve(handler()).catch(()=>{}));
  return el;
}
export function cell(row,key,value,className=""){
  const el=node("td",undefined,className);el.dataset.label=t(key);
  if(value instanceof Node) el.append(value);else el.textContent=String(value ?? "—");
  row.append(el);return el;
}
export function badge(key,online=false){return node("span",t(key),"badge"+(online?" online":""));}
export function empty(target,key,colspan){
  target.replaceChildren();
  const el=node("div",undefined,"empty");
  el.append(node("span","◈","empty-symbol"),node("span",t(key)));
  if(colspan){const row=node("tr"),td=node("td",undefined,"empty");td.colSpan=colspan;td.append(el);row.append(td);target.append(row);}
  else target.append(el);
}
export const time=date=>new Date(date).toLocaleTimeString(locale(),{hour:"2-digit",minute:"2-digit",second:"2-digit"});
let toastTimer;
export function toast(key,args={}){
  $("toast").textContent=t(key,args);$("toast").hidden=false;
  clearTimeout(toastTimer);toastTimer=setTimeout(()=>{$("toast").hidden=true;},4500);
}
export function errorText(error){return t(error.key || "operationFailed");}
export function field(key,value="",options={}){
  const label=node("label"),caption=node("span",t(options.label || key));
  caption.dataset.i18n=options.label || key;
  const input=node("input");input.name=key;input.type=options.type || "text";
  if(input.type==="checkbox"){label.className="check";input.checked=!!value;label.append(input,caption);}
  else {input.value=value ?? "";label.append(caption,input);}
  for(const prop of ["required","min","max","maxLength","inputMode","pattern","autocomplete"]) if(options[prop]!==undefined) input[prop]=options[prop];
  if(input.type==="number") input.step="1";
  return label;
}
export function integer(value,min,max){
  const text=String(value).trim();
  if(!/^\d+$/.test(text) || BigInt(text)<BigInt(min) || BigInt(text)>BigInt(max)) throw new Error("invalidNumber");
  return Number(text);
}
export function changedPatch(original,values){
  const patch={};
  for(const [key,value] of Object.entries(values)){
    if(key==="password"){if(value)patch.password=value;continue;}
    if(String(value)!==String(original[key] ?? ""))patch[key]=value;
  }
  return patch;
}
export async function busy(form,action,errorTarget){
  if(form.dataset.busy)return;
  const controls=[...form.querySelectorAll("button,input,select")];
  const states=controls.map(el=>el.disabled);
  form.dataset.busy="true";form.setAttribute("aria-busy","true");
  controls.forEach(el=>el.disabled=true);
  if(errorTarget)errorTarget.textContent="";
  try {await action();}catch(error){if(error.name!=="AbortError"){if(errorTarget)errorTarget.textContent=errorText(error);else toast(error.key || "operationFailed");}}
  finally {controls.forEach((el,i)=>el.disabled=states[i]);delete form.dataset.busy;form.removeAttribute("aria-busy");}
}

