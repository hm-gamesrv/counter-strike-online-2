import {t,locale,initPreferences,setLanguage,setTheme} from "./i18n.js";
import {request,hasToken,setToken,sessionVersion,accountPath} from "./api.js";
import {$,node,button,cell,badge,empty,time,toast,errorText,busy,integer} from "./ui.js";
import {accountController} from "./accounts.js";

const state={status:null,players:[],accounts:[],activity:[],page:1,updated:null,connected:false,view:"overview"};
let refreshPromise=null,refreshGeneration=-1;
function activity(key,args={},error=false){
  state.activity.unshift({key,args,error,date:Date.now()});
  state.activity=state.activity.slice(0,100);renderActivity();
}
function report(error){
  if(error.name==="AbortError")return;
  activity(error.key || "operationFailed",{},true);
  toast(error.key || "operationFailed");
}
function sessionUI(){
  $("connection").textContent=t(state.connected?"connected":"notConnected");
  $("connection").className="badge"+(state.connected?" online":"");
  $("sessionBtn").textContent=t(hasToken()?"disconnect":"connect");
  $("notice").hidden=state.connected;
  for(const id of ["newAccountBtn","saveBtn","operationsSave"])$(id).disabled=!state.connected;
  $("roomForm").querySelectorAll("input,button").forEach(el=>el.disabled=!state.connected);
}
function reset(){
  state.status=null;state.players=[];state.accounts=[];state.updated=null;state.connected=false;state.page=1;
  $("search").value="";$("token").value="";
  for(const dialog of document.querySelectorAll("dialog"))if(dialog.open)dialog.close();
  render();
}
async function refresh(){
  if(!hasToken())return;
  const generation=sessionVersion();
  if(refreshPromise && refreshGeneration===generation)return refreshPromise;
  refreshGeneration=generation;
  $("refreshBtn").disabled=true;
  const job=(async()=>{
    const [status,players,accounts]=await Promise.all([request("GET","/api/status"),request("GET","/api/players"),request("GET","/api/accounts")]);
    if(generation!==sessionVersion())return;
    Object.assign(state,{status,players:players.players || [],accounts:accounts.accounts || [],connected:true,updated:Date.now()});
    render();
  })();
  refreshPromise=job;
  try {await job;}
  catch(error){if(generation===sessionVersion()){state.connected=false;sessionUI();}throw error;}
  finally{if(refreshPromise===job){refreshPromise=null;$("refreshBtn").disabled=false;}}
}
async function mutate(method,path,body,key,args={}){
  const result=await request(method,path,body);
  activity(key,args);toast(key,args);
  // A completed write is still a success if a subsequent read fails.
  try {await refresh();}catch(error){report(error);}
  return result;
}
const accounts=accountController(mutate,report);
function navigate(){
  state.view=location.hash.slice(1);
  if(!["overview","players","accounts","operations","activity"].includes(state.view))state.view="overview";
  document.querySelectorAll(".view").forEach(el=>el.hidden=el.id!=="view-"+state.view);
  document.querySelectorAll("[data-view]").forEach(el=>{
    el.classList.toggle("active",el.dataset.view===state.view);
    if(el.dataset.view===state.view)el.setAttribute("aria-current","page");else el.removeAttribute("aria-current");
  });
  $("pageTitle").textContent=t(state.view);$("pageDescription").textContent=t(state.view+"Description");
}
function renderStatus(){
  const s=state.status;
  $("metricOnline").textContent=s?.onlineUsers ?? "—";
  $("metricAccounts").textContent=s?.accounts ?? "—";
  $("metricVersion").textContent=s?.version ?? "—";
  $("metricUptime").textContent=s?t("duration",{h:Math.floor(s.uptimeSeconds/3600),m:Math.floor(s.uptimeSeconds%3600/60)}):"—";
  $("metricCapacity").textContent=s?(s.maxUsers===4294967295 || s.maxUsers===0?t("unlimited"):t("capacity",{n:s.maxUsers})):t("waiting");
  $("navCount").textContent=s?.onlineUsers ?? "—";
  $("updatedAt").textContent=state.updated?t("updated",{time:time(state.updated)}):t("waiting");
  $("serverDetails").replaceChildren();
  if(!s){empty($("serverDetails"),"waiting");return;}
  for(const key of ["gamePort","holepunchPort","adminPort","dataDir","userDataDir","shopEnabled","unlockAllWeapons"]){
    const row=node("div"),value=s[key];
    row.append(node("dt",t(key)),node("dd",typeof value==="boolean"?t(value?"on":"off"):value));
    $("serverDetails").append(row);
  }
}
function renderPlayers(){
  $("playerCount").textContent=t("playerCount",{n:state.players.length});
  const rows=$("playerRows");rows.replaceChildren();
  if(!state.connected){empty(rows,"waiting",6);return;}
  if(!state.players.length){empty(rows,"emptyPlayers",6);return;}
  for(const p of state.players){
    const row=node("tr"),identity=node("div");
    identity.append(node("b",p.username),node("small",p.nickname));
    cell(row,"player",identity,"identity");cell(row,"level",p.level);cell(row,"state",badge(p.inGame?"inGame":"idle",p.inGame));
    const location=p.roomId>0?t("channel")+" "+p.channelServerIndex+"/"+p.channelIndex+" · "+t("room")+" "+p.roomId:p.channelIndex>0?t("channel")+" "+p.channelServerIndex+"/"+p.channelIndex:t("lobby");
    cell(row,"location",location);
    cell(row,"roles",[p.gm?"GM":"",p.vip?"VIP":""].filter(Boolean).join(" / ") || "—");
    const actions=node("div",undefined,"row-actions");
    actions.append(button("edit",()=>accounts.editAccount(p.username)),button("addItem",()=>accounts.item(p.username)),button("removeItem",()=>accounts.item(p.username,true)));
    if(!p.gm)actions.append(button("setGM",()=>accounts.confirm("setGM","confirmAction",{name:p.username},()=>mutate("POST",accountPath(p.username)+"/gm",undefined,"gmLog",{name:p.username}))));
    if(!p.vip)actions.append(button("setVIP",()=>mutate("POST",accountPath(p.username)+"/vip",undefined,"vipLog",{name:p.username}).catch(report)));
    actions.append(button("kick",()=>accounts.confirm("kickTitle","kickHint",{name:p.username},()=>mutate("POST","/api/players/"+encodeURIComponent(p.username)+"/kick",undefined,"kickedLog",{name:p.username})),true));
    cell(row,"actions",actions);rows.append(row);
  }
}
function renderAccounts(){
  const query=$("search").value.trim().toLocaleLowerCase();
  const list=state.accounts.filter(a=>[a.username,a.nickname,a.uuid].some(value=>String(value || "").toLocaleLowerCase().includes(query)));
  const pages=Math.max(1,Math.ceil(list.length/20));state.page=Math.min(state.page,pages);
  $("accountCount").textContent=t("resultCount",{n:list.length});
  $("pageInfo").textContent=t("pageInfo",{page:state.page,pages,n:list.length});
  $("prevPage").disabled=state.page<=1;$("nextPage").disabled=state.page>=pages;
  const rows=$("accountRows");rows.replaceChildren();
  if(!state.connected){empty(rows,"waiting",4);return;}
  if(!list.length){empty(rows,state.accounts.length?"noMatches":"emptyAccounts",4);return;}
  for(const a of list.slice((state.page-1)*20,state.page*20)){
    const row=node("tr");cell(row,"username",a.username,"identity");cell(row,"nickname",a.nickname);cell(row,"UUID",a.uuid,"mono muted");
    const actions=node("div",undefined,"row-actions");
    const edit=button("edit",()=>accounts.editAccount(a.username,edit));
    actions.append(edit,button("addItem",()=>accounts.item(a.username)),button("removeItem",()=>accounts.item(a.username,true)),button("delete",()=>accounts.removeAccount(a.username),true));
    cell(row,"actions",actions);rows.append(row);
  }
}
function renderActivity(){
  for(const [id,limit] of [["activityList",100],["recentActivity",4]]){
    const target=$(id);target.replaceChildren();
    if(!state.activity.length){empty(target,"emptyActivity");continue;}
    for(const entry of state.activity.slice(0,limit)){
      const row=node("div",undefined,"activity-row"+(entry.error?" error":""));
      const timestamp=node("time",time(entry.date));timestamp.dateTime=new Date(entry.date).toISOString();
      row.append(node("span",entry.error?"!":"✓","activity-symbol"),node("p",t(entry.key,entry.args)),timestamp);target.append(row);
    }
  }
}
function render(){navigate();sessionUI();renderStatus();renderPlayers();renderAccounts();renderActivity();}
function showLogin(){$("loginError").textContent="";if(!$("loginDialog").open)$("loginDialog").showModal();}
$("loginForm").addEventListener("submit",event=>{
  event.preventDefault();const token=$("token").value.trim();
  busy($("loginForm"),async()=>{
    setToken(token);
    try{await refresh();activity("connectedLog");$("loginDialog").close();$("token").value="";}
    catch(error){setToken("");state.connected=false;sessionUI();throw error;}
  },$("loginError"));
});
$("sessionBtn").addEventListener("click",()=>{
  if(hasToken()){setToken("");reset();activity("disconnectedLog");}
  else showLogin();
});
$("openLogin").addEventListener("click",showLogin);
$("refreshBtn").addEventListener("click",()=>refresh().catch(report));
for(const id of ["saveBtn","operationsSave"])$(id).addEventListener("click",()=>{
  busy($(id).parentElement,()=>mutate("POST","/api/save",undefined,"savedLog"));
});
$("newAccountBtn").addEventListener("click",accounts.newAccount);
$("search").addEventListener("input",()=>{state.page=1;renderAccounts();});
$("prevPage").addEventListener("click",()=>{state.page--;renderAccounts();});
$("nextPage").addEventListener("click",()=>{state.page++;renderAccounts();});
$("clearActivity").addEventListener("click",()=>{state.activity=[];renderActivity();});
$("language").addEventListener("change",event=>{setLanguage(event.target.value);render();accounts.translateDialogs();});
$("theme").addEventListener("change",event=>setTheme(event.target.value));
$("roomForm").addEventListener("submit",event=>{
  event.preventDefault();
  try {
    const form=$("roomForm").elements,args={server:integer(form.server.value,1,255),channel:integer(form.channel.value,1,255),id:integer(form.room.value,1,65535)};
    accounts.confirm("closeTitle","closeHint",args,()=>mutate("DELETE","/api/rooms/"+args.server+"/"+args.channel+"/"+args.id,undefined,"roomClosedLog",args));
  }catch{toast("invalidNumber");}
});
document.querySelectorAll("[data-go]").forEach(el=>el.addEventListener("click",()=>{location.hash=el.dataset.go;}));
document.querySelectorAll("[data-close]").forEach(el=>el.addEventListener("click",()=>{$(el.dataset.close).close();}));
document.querySelectorAll("dialog").forEach(dialog=>{
  dialog.addEventListener("cancel",event=>{if(dialog.querySelector("[data-busy]"))event.preventDefault();});
  dialog.addEventListener("close",()=>{dialog.querySelectorAll('input[type="password"]').forEach(input=>input.value="");});
});
document.addEventListener("session-expired",()=>{reset();showLogin();$("loginError").textContent=t("authError");});
window.addEventListener("hashchange",navigate);
initPreferences();render();
if(hasToken())refresh().catch(report);

