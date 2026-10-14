import {test} from "node:test";
import assert from "node:assert/strict";
import {readFile} from "node:fs/promises";
const memory=new Map();
globalThis.window={sessionStorage:{getItem:key=>memory.get(key),setItem:(key,value)=>memory.set(key,value),removeItem:key=>memory.delete(key)},localStorage:{getItem:()=>null}};
globalThis.document={dispatchEvent:()=>{}};
const {request,setToken,hasToken,APIError}=await import("../web/api.js");
const {changedPatch,integer}=await import("../web/ui.js");
const {dictionaries}=await import("../web/i18n.js");
const token="unit_test_token_2026";

test("editing a nickname does not overwrite unchanged balances or false/zero values",()=>{
 const original={nickname:"Alice",points:"18446744073709551615",cash:15,gm:true,vipLevel:2};
 assert.deepEqual(changedPatch(original,{nickname:"New Alice",points:"18446744073709551615",cash:15,gm:false,vipLevel:0,password:""}),{nickname:"New Alice",gm:false,vipLevel:0});
});
test("integer validation rejects truncation, negative values and overflow",()=>{
 for(const value of ["1.2","2e3","-1","NaN","65536",""])assert.throws(()=>integer(value,0,65535));
 assert.equal(integer("65535",0,65535),65535);
});
test("API preserves uint64 points and sends credentials only in headers",async()=>{
 setToken(token);
 globalThis.fetch=async(path,opts)=>{
   assert.equal(path,"/api/accounts/alice");
   assert.equal(opts.headers.Authorization,"Bearer "+token);
   assert.equal(opts.body,'{"points":18446744073709551615,"nickname":"Alice"}');
   return new Response('{"points":18446744073709551615,"nickname":"Alice"}');
 };
 const result=await request("PATCH","/api/accounts/alice",{points:"18446744073709551615",nickname:"Alice"});
 assert.equal(result.points,"18446744073709551615");
});
test("expired credentials are removed from session storage",async()=>{
 setToken(token);globalThis.fetch=async()=>new Response('{"error":"bad token"}',{status:401});
 await assert.rejects(request("GET","/api/status"),error=>error instanceof APIError && error.key==="authError");
 assert.equal(hasToken(),false);assert.equal(memory.has("cso2-admin-token"),false);
});
test("signing out prevents a late response from restoring a previous session",async()=>{
 setToken(token);
 let resolve;
 globalThis.fetch=()=>new Promise(done=>{resolve=done;});
 const pending=request("GET","/api/status");setToken("");
 resolve(new Response('{"accounts":12}'));
 await assert.rejects(pending,error=>error.name==="AbortError");
 assert.equal(hasToken(),false);
});
test("network errors produce a translated error key",async()=>{
 setToken(token);globalThis.fetch=async()=>{throw new TypeError("fetch failed");};
 await assert.rejects(request("GET","/api/status"),error=>error.key==="networkError");
});
test("both dictionaries cover all declared UI strings",async()=>{
 assert.deepEqual(Object.keys(dictionaries.zh).sort(),Object.keys(dictionaries.en).sort());
 const html=await readFile(new URL("../page.html",import.meta.url),"utf8");
 for(const match of html.matchAll(/data-(?:i18n|aria|placeholder)="([^"]+)"/g)){
  assert.ok(dictionaries.zh[match[1]],match[1]);assert.ok(dictionaries.en[match[1]],match[1]);
 }
});

