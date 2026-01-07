import{b as U,g as B,j as k,a2 as I,h as r,f as N,a3 as D,a4 as X,a5 as z,y as u,o as c,w as t,a as o,k as y,t as m,l as e,q as w,p as S,s as H,x as j,n as A,B as R}from"./index-CSEZG5G6.js";import{M as G}from"./preview-CpAvmqHn.js";import{t as O}from"./tools-BOWg2iRh.js";import{_ as $}from"./_plugin-vue_export-helper-DlAUqK2U.js";import{c as K,a as J,V as F}from"./VCard-BL1j-OcQ.js";import{V as d}from"./VRow-Da0Ja_x_.js";import{V as Q}from"./VAlert-CEvD-mRx.js";import{V as W}from"./VSelect-D0jky2UU.js";import{V as Y}from"./VTextField-IaxNnzhY.js";import"./axios-B1uS2EtD.js";/* empty css              */import"./VInput-DzbtYzdA.js";import"./index-BTquysF0.js";import"./VList-DWSnpUg8.js";import"./ssrBoot-CO9u9Llw.js";import"./VMenu-DrVjG3xF.js";import"./dialog-transition-Cta2BUts.js";import"./VCheckboxBtn-C1zLx64S.js";import"./VSelectionControl-CzrMTBrL.js";import"./VChip-DXlk2YaI.js";const Z={class:"card-header"},ee={__name:"token",setup(ne){const{t:n}=U(),g=B(),_=k(()=>g.theme),L=k(()=>I(g.language)),i=r({expiration:void 0}),V=[{title:n("tools.token.select.day"),value:24},{title:n("tools.token.select.week"),value:168},{title:n("tools.token.select.month"),value:720},{title:n("tools.token.select.year"),value:365*24},{title:n("tools.token.select.permanent"),value:0}];r(!1);const a=r(""),x=()=>{if(i.value.expiration===void 0){R(n("tools.token.noSelected"),"error");return}O.token.post(i.value).then(l=>{a.value=l.data,i.value.expiration=void 0,R(l.message,"success")})},C=r(`\`\`\`python [id:Python]
import requests

url = "http://{ip}:{port}"
token = "your token"
# 中文
lang = "zh"
# English
# lang = "en"

payload = {}
headers = {
    'X-DMP-TOKEN': token,
    'X-I18n-Lang': lang
}

response = requests.request("GET", url, headers=headers, data=payload)

print(response.text)
\`\`\``),q=r(`\`\`\`golang [id:Golang]
package main

import (
  "fmt"
  "net/http"
  "io"
)

func main() {
  token := "your token"
  url := "http://{ip}:{port}"
  method := "GET"
  //中文
  lang := "zh"
  //English
  //lang := "en"

  client := &http.Client{}
  req, err := http.NewRequest(method, url, nil)

  if err != nil {
    fmt.Println(err)
    return
  }
  req.Header.Add("X-DMP-TOKEN", token)
  req.Header.Add("X-I18n-Lang", lang)

  res, err := client.Do(req)
  if err != nil {
    fmt.Println(err)
    return
  }
  defer res.Body.Close()

  body, err := io.ReadAll(res.Body)
  if err != nil {
    fmt.Println(err)
    return
  }
  fmt.Println(string(body))
}
\`\`\``),b=r(`\`\`\`java [id:Java]
import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

public class Main {
    public static void main(String[] args) {
        try {
            // 定义请求的 URL
            String url = "http://{ip}:{port}";
            // 定义 token 和语言
            String token = "your token";
            String lang = "zh"; // 中文
            // String lang = "en"; // English

            // 创建 URL 对象
            URL apiUrl = new URL(url);
            // 打开连接
            HttpURLConnection connection = (HttpURLConnection) apiUrl.openConnection();
            // 设置请求方法
            connection.setRequestMethod("GET");
            // 添加请求头
            connection.setRequestProperty("X-DMP-TOKEN", token);
            connection.setRequestProperty("X-I18n-Lang", lang);

            // 获取响应码
            int responseCode = connection.getResponseCode();
            System.out.println("Response Code: " + responseCode);

            // 读取响应内容
            BufferedReader in = new BufferedReader(new InputStreamReader(connection.getInputStream()));
            String inputLine;
            StringBuilder response = new StringBuilder();

            while ((inputLine = in.readLine()) != null) {
                response.append(inputLine);
            }
            in.close();

            // 打印响应内容
            System.out.println("Response Body: " + response.toString());
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
\`\`\``),E=r("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),P=r(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),T=C.value+`

`+q.value+`

`+b.value+`

`+E.value+`

`+P.value,h=r(window.innerHeight),f=D(()=>{h.value=window.innerHeight},200),v=()=>Math.max(2,Math.floor(h.value-150));return N(async()=>{window.addEventListener("resize",f)}),X(()=>{window.removeEventListener("resize",f)}),(l,s)=>{const M=z("copy");return c(),u(F,{height:v()},{default:t(()=>[o(K,null,{default:t(()=>[y("div",Z,[y("span",null,m(e(n)("tools.token.title")),1)])]),_:1}),o(J,{class:"mx-2"},{default:t(()=>[o(d,{class:"mt-4"},{default:t(()=>[o(Q,{color:"warning",density:"compact"},{default:t(()=>[w(m(e(n)("tools.token.tip")),1)]),_:1})]),_:1}),e(a)===""?(c(),u(d,{key:0,class:"mt-8 d-flex align-center"},{default:t(()=>[o(W,{modelValue:e(i).expiration,"onUpdate:modelValue":s[0]||(s[0]=p=>e(i).expiration=p),label:e(n)("tools.token.select.label"),items:V},null,8,["modelValue","label"]),o(S,{size:"large",class:"ml-4",onClick:x},{default:t(()=>[w(m(e(n)("tools.token.create")),1)]),_:1})]),_:1})):(c(),u(d,{key:1,class:"mt-8"},{default:t(()=>[o(Y,{modelValue:e(a),"onUpdate:modelValue":s[1]||(s[1]=p=>H(a)?a.value=p:null)},{"append-inner":t(()=>[j(o(S,{variant:"text",icon:"ri-file-copy-line"},null,512),[[M,e(a)]])]),_:1},8,["modelValue"])]),_:1})),o(d,{class:"mt-8"},{default:t(()=>[o(e(G),{"model-value":T,theme:e(_),language:e(L),"preview-theme":"github",class:"mdp",style:A({"overflow-y":"auto",height:v()-220+"px"})},null,8,["theme","language","style"])]),_:1})]),_:1})]),_:1},8,["height"])}}},Re=$(ee,[["__scopeId","data-v-fdfa2800"]]);export{Re as default};
