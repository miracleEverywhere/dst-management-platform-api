import{h as n}from"./index-CS37_vV1.js";const a={osInfo:{url:"/tools/os_info",get:async function(t){return await n.get(this.url,t)}},install:{url:"/tools/install",post:async function(t){return await n.post(this.url,t)}},installStatus:{url:"/tools/install/status",get:async function(t){return await n.get(this.url,t)}},announce:{url:"/tools/announce",get:async function(t){return await n.get(this.url,t)},post:async function(t){return await n.post(this.url,t)},delete:async function(t){return await n.delete(this.url,t)},put:async function(t){return await n.put(this.url,t)}},update:{url:"/tools/update",get:async function(t){return await n.get(this.url,t)},put:async function(t){return await n.put(this.url,t)}},backup:{url:"/tools/backup",get:async function(t){return await n.get(this.url,t)},post:async function(t){return await n.post(this.url,t)},put:async function(t){return await n.put(this.url,t)},delete:async function(t){return await n.delete(this.url,t)}},backupRestore:{url:"/tools/backup/restore",post:async function(t){return await n.post(this.url,t)}},backupDownload:{url:"/tools/backup/download",post:async function(t){return await n.post(this.url,t)}},multiDelete:{url:"/tools/backup/multi",delete:async function(t){return await n.delete(this.url,t)}},mod:{install:{all:{url:"/tools/mod/install/all",post:async function(t){return await n.post(this.url,t)}}}},statistics:{url:"/tools/statistics",get:async function(t){return await n.get(this.url,t)}},keepalive:{url:"/tools/keepalive",get:async function(t){return await n.get(this.url,t)},put:async function(t){return await n.put(this.url,t)}},replaceSo:{url:"/tools/replace_so",post:async function(t){return await n.post(this.url,t)}},token:{create:{url:"/tools/token",post:async function(t){return await n.post(this.url,t)}}}};export{a as t};