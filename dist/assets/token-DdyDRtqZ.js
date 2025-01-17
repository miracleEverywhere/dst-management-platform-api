import{u as b,a as D,g as S,c as B,o as M,r as h,B as r,C as q,d as v,e as g,f as n,w as p,i as e,t as o,j as t,m as c,aj as E,l as N,ak as Y,n as j,X as z,$ as U,y as A}from"./index-DLH8rKkR.js";import{s as I}from"./index-Db3lJbf9.js";const X={class:"page-div"},$={class:"card-header"},F={style:{display:"flex","align-items":"center"}},G={key:0},L={class:"tip custom-block"},O={style:{"font-weight":"bolder"}},R={style:{"margin-top":"5px"}},H={style:{"margin-top":"20px"}},J=z({name:"toolsToken"}),W=Object.assign(J,{setup(K){const{t:s}=b(),{isMobile:k}=D(),y=S(),f=B(()=>y.isDark);M(async()=>{l.value.expiredTime=new Date().getTime()});const l=h({expiredTime:0}),i=h(""),x=()=>{U.token.create.post(l.value).then(u=>{i.value=u.data,A(u.message)})},m=`import requests

url = "http://{ip}:{port}"
token = "your token"
# 中文
lang = "zh"
# English
# lang = "en"

payload = {}
headers = {
    'Authorization': token,
    'X-I18n-Lang': lang
}

response = requests.request("GET", url, headers=headers, data=payload)

print(response.text)`;return(u,a)=>{const _=r("el-button"),V=r("el-date-picker"),T=r("el-input"),w=r("el-card"),C=q("copy");return v(),g("div",X,[n(w,{shadow:"never",style:{height:"80vh"}},{header:p(()=>[e("div",$,[e("span",null,o(t(s)("tools.token.title")),1),n(_,{type:"primary",onClick:x},{default:p(()=>[c(o(t(s)("tools.token.createButton")),1)]),_:1})])]),default:p(()=>[e("div",null,[e("div",F,[e("span",null,o(t(s)("tools.token.expiredTime")),1),n(V,{modelValue:l.value.expiredTime,"onUpdate:modelValue":a[0]||(a[0]=d=>l.value.expiredTime=d),format:"YYYY-MM-DD",size:"large",style:{width:"160px","margin-left":"5px"},type:"date","value-format":"x"},null,8,["modelValue"])]),i.value?(v(),g("div",G,[e("div",L,[e("div",null,[c(o(t(s)("tools.token.tip.tip1"))+" ",1),e("span",O,o(t(E)(l.value.expiredTime)),1),c(" "+o(t(s)("tools.token.tip.tip2")),1)]),e("div",R,o(t(s)("tools.token.tip.tip3")),1)]),n(T,{modelValue:i.value,"onUpdate:modelValue":a[1]||(a[1]=d=>i.value=d),style:{"max-width":"100%"}},{append:p(()=>[N(n(_,{icon:t(Y)},null,8,["icon"]),[[C,i.value]])]),_:1},8,["modelValue"]),e("div",H,[e("div",null,o(t(s)("tools.token.usage")),1),n(I,{ref:"twoCodeRef",modelValue:m,"onUpdate:modelValue":a[2]||(a[2]=d=>m=d),height:t(k)?200:400,"read-only":!0,theme:f.value?"darcula":"idea",mode:"python",style:{"margin-top":"10px"}},null,8,["height","theme"])])])):j("",!0)])]),_:1})])}}});export{W as default};
