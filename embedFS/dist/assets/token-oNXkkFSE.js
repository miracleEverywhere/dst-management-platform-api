import{_ as U,b as B,g as I,i as k,a7 as N,f as a,j as z,a8 as D,a9 as X,aa as H,A as u,o as c,w as t,a as o,ab as A,k as y,t as m,l as e,p as G,v as w,s as S,x as j,z as O,n as $,V as K,G as R}from"./index-530dAX-D.js";import{M as J}from"./preview-DX0uKi2w.js";import{t as F}from"./tools-CkgHcYXF.js";import{V as d}from"./VRow-CBgutLCV.js";import{V as Q}from"./VAlert-CG6IosLD.js";import{V as W}from"./VSelect-IUSvuXCj.js";import{V as Y}from"./VTextField-Chaz3BK2.js";import"./axios-D8jNJj4u.js";import"./VInput-UHNs9aN_.js";import"./index-hDlxYiYz.js";import"./filter-Bgu_XV2s.js";import"./VSelectionControl-DzDoyQyy.js";import"./VList-DMXRTjF_.js";import"./ssrBoot-Bhm-MG3N.js";import"./VMenu-BQ45tOsV.js";import"./dialog-transition-Bw_FXxS3.js";import"./VChip-BkxZKIOe.js";const Z={class:"card-header"},ee={__name:"token",setup(ne){const{t:n}=B(),g=I(),_=k(()=>g.theme),L=k(()=>N(g.language)),l=a({expiration:void 0}),V=[{title:n("tools.token.select.day"),value:24},{title:n("tools.token.select.week"),value:168},{title:n("tools.token.select.month"),value:720},{title:n("tools.token.select.year"),value:365*24},{title:n("tools.token.select.permanent"),value:0}];a(!1);const r=a(""),x=()=>{if(l.value.expiration===void 0){R(n("tools.token.noSelected"),"error");return}F.token.post(l.value).then(s=>{r.value=s.data,l.value.expiration=void 0,R(s.message,"success")})},C=a(`\`\`\`python [id:Python]
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
\`\`\``),q=a(`\`\`\`golang [id:Golang]
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
\`\`\``),b=a(`\`\`\`java [id:Java]
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
\`\`\``),E=a("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),P=a(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),T=C.value+`

`+q.value+`

`+b.value+`

`+E.value+`

`+P.value,h=a(window.innerHeight),f=D(()=>{h.value=window.innerHeight},200),v=()=>Math.max(2,Math.floor(h.value-150));return z(async()=>{window.addEventListener("resize",f)}),X(()=>{window.removeEventListener("resize",f)}),(s,i)=>{const M=H("copy");return c(),u(K,{height:v()},{default:t(()=>[o(A,null,{default:t(()=>[y("div",Z,[y("span",null,m(e(n)("tools.token.title")),1)])]),_:1}),o(G,{class:"mx-2"},{default:t(()=>[o(d,{class:"mt-4"},{default:t(()=>[o(Q,{color:"warning",density:"compact"},{default:t(()=>[w(m(e(n)("tools.token.tip")),1)]),_:1})]),_:1}),e(r)===""?(c(),u(d,{key:0,class:"mt-8 d-flex align-center"},{default:t(()=>[o(W,{modelValue:e(l).expiration,"onUpdate:modelValue":i[0]||(i[0]=p=>e(l).expiration=p),label:e(n)("tools.token.select.label"),items:V},null,8,["modelValue","label"]),o(S,{size:"large",class:"ml-4",onClick:x},{default:t(()=>[w(m(e(n)("tools.token.create")),1)]),_:1})]),_:1})):(c(),u(d,{key:1,class:"mt-8"},{default:t(()=>[o(Y,{modelValue:e(r),"onUpdate:modelValue":i[1]||(i[1]=p=>j(r)?r.value=p:null)},{"append-inner":t(()=>[O(o(S,{variant:"text",icon:"ri-file-copy-line"},null,512),[[M,e(r)]])]),_:1},8,["modelValue"])]),_:1})),o(d,{class:"mt-8"},{default:t(()=>[o(e(J),{"model-value":T,theme:e(_),language:e(L),"preview-theme":"github",class:"mdp",style:$({"overflow-y":"auto",height:v()-220+"px"})},null,8,["theme","language","style"])]),_:1})]),_:1})]),_:1},8,["height"])}}},ye=U(ee,[["__scopeId","data-v-fdfa2800"]]);export{ye as default};
