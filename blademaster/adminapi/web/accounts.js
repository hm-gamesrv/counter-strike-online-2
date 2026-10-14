import {t,translate} from "./i18n.js";
import {request,accountPath,sessionVersion,APIError} from "./api.js";
import {$,node,field,badge,changedPatch,integer,busy} from "./ui.js";

export function accountController(mutate,report){
  let original=null,itemTarget=null,confirmAction=null,confirmConfig=null;
  const values=()=>new FormData($("accountForm"));
  function populate(account){
    original=account;
    $("accountError").textContent="";
    $("accountSummary").replaceChildren();
    const fields=$("accountFields");fields.replaceChildren();
    $("accountDialogTitle").textContent=t(account?"edit":"newAccount");
    $("editHint").dataset.i18n=account?"editHint":"nameLimit";
    $("editHint").textContent=t(account?"editHint":"nameLimit");
    if(account){
      $("accountSummary").append(node("b",account.username),node("p",account.uuid,"muted"),badge(account.online?"online":"offline",account.online));
      const stats=node("div",undefined,"summary-grid");
      const summary={matches:account.playedMatches+" / "+account.wins,combat:account.kills+" / "+account.deaths+" / "+account.assists,headshots:account.headshots,itemCount:account.itemCount};
      for(const [key,value] of Object.entries(summary)){const box=node("div");const title=node("small",t(key));title.dataset.i18n=key;box.append(title,node("b",value));stats.append(box);}
      $("accountSummary").append(stats);
    }else fields.append(field("username","",{required:true,maxLength:64,autocomplete:"off"}));
    fields.append(field("nickname",account?.nickname,{required:true,maxLength:64,autocomplete:"off"}));
    if(!account)fields.append(field("password","",{type:"password",required:true,autocomplete:"new-password"}));
    fields.append(field("mail",account?.mail,{type:"email",autocomplete:"off"}));
    if(account){
      for(const [key,max] of Object.entries({level:65535,points:"18446744073709551615",cash:4294967295,mpoints:4294967295,vipLevel:6})){
        fields.append(field(key,account[key],key==="points" ? {required:true,inputMode:"numeric",pattern:"[0-9]+"}:{type:"number",required:true,min:0,max}));
      }
      fields.append(field("password","",{type:"password",autocomplete:"new-password",label:"newPassword"}));
    }
    fields.append(field("gm",account?.gm,{type:"checkbox"}));
    if(!account)fields.append(field("vip",false,{type:"checkbox"}));
  }
  function validateName(value){
    const length=new TextEncoder().encode(value).length;
    if(!length || length>64)throw new APIError("invalidName");
  }
  function newAccount(){
    populate(null);$("accountDialog").showModal();
  }
  async function editAccount(name,trigger){
    if(trigger)trigger.disabled=true;
    const version=sessionVersion();
    try {
      const account=await request("GET",accountPath(name));
      if(version!==sessionVersion())return;
      populate(account);$("accountDialog").showModal();
    }catch(error){report(error);}finally{if(trigger)trigger.disabled=false;}
  }
  function confirm(title,description,args,action){
    confirmConfig={title,description,args};confirmAction=action;
    $("confirmTitle").textContent=t(title,args);$("confirmDescription").textContent=t(description,args);
    $("confirmError").textContent="";$("confirmDialog").showModal();
  }
  function removeAccount(name){
    confirm("deleteTitle","deleteHint",{name},async()=>{await mutate("DELETE",accountPath(name),undefined,"deletedLog",{name});});
  }
  function item(name,remove=false){
    itemTarget={name,remove};
    $("itemForm").reset();$("itemError").textContent="";
    $("itemTitle").textContent=t(remove?"removeItem":"addItem");
    $("itemAccount").textContent=name;
    $("quantityField").hidden=remove;
    $("itemForm").elements.count.disabled=remove;
    $("itemDialog").showModal();
  }
  $("accountForm").addEventListener("submit",event=>{
    event.preventDefault();
    let body;
    try {
      const data=values();
      body={nickname:String(data.get("nickname")).trim(),mail:String(data.get("mail")).trim(),gm:data.has("gm"),password:String(data.get("password"))};
      validateName(body.nickname);
      if(original){
        for(const [key,max] of Object.entries({level:65535,cash:4294967295,mpoints:4294967295,vipLevel:6}))body[key]=integer(data.get(key),0,max);
        const points=String(data.get("points")).trim();
        if(!/^\d+$/.test(points) || BigInt(points)>18446744073709551615n)throw new APIError("invalidNumber");
        body.points=BigInt(points).toString();
        body=changedPatch(original,body);
      }else{
        body.username=String(data.get("username")).trim();validateName(body.username);body.vip=data.has("vip");
      }
    }catch(error){$("accountError").textContent=t(error.key || error.message);return;}
    if(!Object.keys(body).length){$("accountError").textContent=t("noChanges");return;}
    const name=original?.username || body.username,editing=!!original;
    busy($("accountForm"),async()=>{
      await mutate(editing?"PATCH":"POST",editing?accountPath(name):"/api/accounts",body,editing?"editedLog":"createdLog",{name});
      $("accountDialog").close();$("accountForm").reset();original=null;
    },$("accountError"));
  });
  $("itemForm").addEventListener("submit",event=>{
    event.preventDefault();
    let id,count;
    try {id=integer($("itemForm").elements.itemId.value,1,4294967295);count=itemTarget.remove?1:integer($("itemForm").elements.count.value,1,65535);}
    catch {$("itemError").textContent=t("invalidNumber");return;}
    const {name,remove}=itemTarget;
    const action=async()=>{
      await mutate(remove?"DELETE":"POST",accountPath(name)+"/items"+(remove?"/"+id:""),remove?undefined:{itemId:id,count},remove?"itemRemovedLog":"itemAddedLog",{name,id,count});
      $("itemDialog").close();
    };
    if(remove)confirm("removeTitle","removeHint",{id,name},action);
    else busy($("itemForm"),action,$("itemError"));
  });
  $("confirmForm").addEventListener("submit",event=>{
    event.preventDefault();
    busy($("confirmForm"),async()=>{await confirmAction();$("confirmDialog").close();},$("confirmError"));
  });
  function translateDialogs(){
    if($("accountDialog").open)$("accountDialogTitle").textContent=t(original?"edit":"newAccount");
    if($("itemDialog").open)$("itemTitle").textContent=t(itemTarget.remove?"removeItem":"addItem");
    if($("confirmDialog").open){$("confirmTitle").textContent=t(confirmConfig.title,confirmConfig.args);$("confirmDescription").textContent=t(confirmConfig.description,confirmConfig.args);}
    document.querySelectorAll("dialog").forEach(dialog=>translate(dialog));
  }
  return {newAccount,editAccount,removeAccount,item,confirm,translateDialogs};
}

