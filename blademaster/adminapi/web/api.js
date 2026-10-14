import {storage} from "./i18n.js";
const TOKEN_KEY="cso2-admin-token";
let token=storage("sessionStorage",TOKEN_KEY) || "";
let generation=0;
const requests=new Set();
export class APIError extends Error {
  constructor(key,status=0,detail=""){super(key);this.key=key;this.status=status;this.detail=detail;}
}
export const hasToken=()=>!!token;
export const sessionVersion=()=>generation;
export function setToken(value){
  generation++;
  for(const controller of requests) controller.abort();
  requests.clear();
  token=value.trim();
  storage("sessionStorage",TOKEN_KEY,token || null);
}
export async function request(method,path,body){
  if(!token) throw new APIError("authError",401);
  const version=generation, controller=new AbortController();
  requests.add(controller);
  const timeout=setTimeout(()=>controller.abort(),15000);
  try {
    const headers={Authorization:"Bearer "+token};
    let payload;
    if(body!==undefined){
      headers["Content-Type"]="application/json";
      // Points are uint64 on the server. Emit their decimal representation without
      // passing through a lossy JavaScript Number.
      if(typeof body.points==="string" && !/^\d+$/.test(body.points)) throw new APIError("invalidNumber");
      payload=JSON.stringify(body).replace(/"points":"(\d+)"/g,'"points":$1');
    }
    const response=await fetch(path,{method,headers,body:payload,cache:"no-store",signal:controller.signal,credentials:"omit"});
    const raw=await response.text();
    if(version!==generation) throw new DOMException("Session changed","AbortError");
    let data;
    try { data=raw ? JSON.parse(raw.replace(/("points"\s*:\s*)(\d+)(?=\s*[,}])/g,'$1"$2"')) : null; }
    catch { throw new APIError("serverError",response.status); }
    if(!response.ok){
      const key=({400:"badRequest",401:"authError",404:"notFound",409:"conflict"})[response.status] || "serverError";
      if(response.status===401){setToken("");document.dispatchEvent(new Event("session-expired"));}
      throw new APIError(key,response.status,data?.error || "");
    }
    return data;
  } catch(error){
    if(version!==generation && error.status!==401) throw new DOMException("Session changed","AbortError");
    if(error instanceof APIError) throw error;
    throw new APIError(error.name==="AbortError" ? "timeout":"networkError");
  } finally {clearTimeout(timeout);requests.delete(controller);}
}
export const accountPath=name=>"/api/accounts/"+encodeURIComponent(name);

