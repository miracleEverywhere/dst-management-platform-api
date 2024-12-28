import{t as C}from"./index-BXFs1hAn.js";import{u as D,a as S,g as B,c as M,o as q,r as h,A as d,B as E,d as v,e as g,f as n,w as p,i as e,t as o,j as t,m as c,l as N,ab as Y,n as z,V as A,x as U}from"./index-CJaPNTvg.js";import{t as j}from"./tools-nSbkALEa.js";import{s as I}from"./index-BciGCvPL.js";const F={class:"page-div"},G={class:"card-header"},L={style:{display:"flex","align-items":"center"}},O={key:0},R={class:"tip custom-block"},X={style:{"font-weight":"bolder"}},$={style:{"margin-top":"5px"}},H={style:{"margin-top":"20px"}},J=A({name:"toolsToken"}),ee=Object.assign(J,{setup(K){const{t:s}=D(),{isMobile:k}=S(),y=B(),f=M(()=>y.isDark);q(async()=>{l.value.expiredTime=new Date().getTime()});const l=h({expiredTime:0}),i=h(""),x=()=>{C.token.create.post(l.value).then(u=>{i.value=u.data,U(u.message)})},m=`import requests

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

print(response.text)`;return(u,a)=>{const _=d("el-button"),V=d("el-date-picker"),T=d("el-input"),b=d("el-card"),w=E("copy");return v(),g("div",F,[n(b,{shadow:"never",style:{height:"80vh"}},{header:p(()=>[e("div",G,[e("span",null,o(t(s)("tools.token.title")),1),n(_,{type:"primary",onClick:x},{default:p(()=>[c(o(t(s)("tools.token.createButton")),1)]),_:1})])]),default:p(()=>[e("div",null,[e("div",L,[e("span",null,o(t(s)("tools.token.expiredTime")),1),n(V,{modelValue:l.value.expiredTime,"onUpdate:modelValue":a[0]||(a[0]=r=>l.value.expiredTime=r),format:"YYYY-MM-DD",size:"large",style:{width:"160px","margin-left":"5px"},type:"date","value-format":"x"},null,8,["modelValue"])]),i.value?(v(),g("div",O,[e("div",R,[e("div",null,[c(o(t(s)("tools.token.tip.tip1"))+" ",1),e("span",X,o(t(j)(l.value.expiredTime)),1),c(" "+o(t(s)("tools.token.tip.tip2")),1)]),e("div",$,o(t(s)("tools.token.tip.tip3")),1)]),n(T,{modelValue:i.value,"onUpdate:modelValue":a[1]||(a[1]=r=>i.value=r),style:{"max-width":"100%"}},{append:p(()=>[N(n(_,{icon:t(Y)},null,8,["icon"]),[[w,i.value]])]),_:1},8,["modelValue"]),e("div",H,[e("div",null,o(t(s)("tools.token.usage")),1),n(I,{ref:"twoCodeRef",modelValue:m,"onUpdate:modelValue":a[2]||(a[2]=r=>m=r),height:t(k)?200:400,"read-only":!0,theme:f.value?"darcula":"idea",mode:"python",style:{"margin-top":"10px"}},null,8,["height","theme"])])])):z("",!0)])]),_:1})])}}});export{ee as default};
