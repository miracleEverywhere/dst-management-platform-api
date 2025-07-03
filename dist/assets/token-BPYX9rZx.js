import{t as A}from"./index-D0yhoPar.js";import{u as E,a as I,g as V,c as k,o as z,r as l,K as r,L as M,f as m,i as g,k as n,l as i,m as o,t as u,j as e,q as j,p as D,ax as G,a1 as $,D as H,B as N}from"./index-BSCyfger.js";import{M as X}from"./preview-DlzrvJ7g.js";const J={class:"page-div"},O={class:"card-header"},F={style:{display:"flex"}},K={key:0},Q={class:"tip custom-block"},W={style:{"margin-top":"5px"}},Y={style:{"margin-top":"20px"}},Z={key:1,style:{height:"60vh"},class:"fcc"},ee=$({name:"toolsToken"}),re=Object.assign(ee,{setup(ne){const{t}=E();I();const v=V(),y=k(()=>v.language),f=k(()=>v.isDark);z(async()=>{});const s=l({expiredTime:null}),a=l(""),S=()=>{if(s.value.expiredTime===null){H(y.value==="zh"?"请选择过期时间":"Please select expire time");return}A.token.create.post(s.value).then(c=>{a.value=c.data,N(c.message)})},R=l(`\`\`\`python [id:Python]
import requests

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

print(response.text)
\`\`\``),b=l(`\`\`\`golang [id:Golang]
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
  req.Header.Add("Authorization", token)
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
\`\`\``),w=l(`\`\`\`java [id:Java]
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
            connection.setRequestProperty("Authorization", token);
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
\`\`\``),L=l("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'Authorization: token' \\\n--header 'X-I18n-Lang: lang'\n```"),q=l(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("Authorization", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-Json
\`\`\``),x=R.value+`

`+b.value+`

`+w.value+`

`+L.value+`

`+q.value;return(c,p)=>{const d=r("el-option"),C=r("el-select"),_=r("el-button"),T=r("el-input"),B=r("el-result"),U=r("el-card"),P=M("copy");return m(),g("div",J,[n(U,{shadow:"never",style:{"min-height":"80vh"}},{header:i(()=>[o("div",O,[o("span",null,u(e(t)("tools.token.title")),1),o("div",F,[n(C,{modelValue:s.value.expiredTime,"onUpdate:modelValue":p[0]||(p[0]=h=>s.value.expiredTime=h),placeholder:e(t)("tools.token.expiredTime"),style:{width:"20vw","margin-right":"20px","font-weight":"lighter"}},{default:i(()=>[n(d,{label:e(t)("tools.token.options.day"),value:24},null,8,["label"]),n(d,{label:e(t)("tools.token.options.month"),value:720},null,8,["label"]),n(d,{label:e(t)("tools.token.options.year"),value:8760},null,8,["label"]),n(d,{label:e(t)("tools.token.options.forever"),value:1752e3},null,8,["label"])]),_:1},8,["modelValue","placeholder"]),n(_,{type:"primary",onClick:S},{default:i(()=>[j(u(e(t)("tools.token.createButton")),1)]),_:1})])])]),default:i(()=>[o("div",null,[a.value?(m(),g("div",K,[o("div",Q,[o("div",W,u(e(t)("tools.token.tip.tip3")),1)]),n(T,{modelValue:a.value,"onUpdate:modelValue":p[1]||(p[1]=h=>a.value=h),style:{"max-width":"100%"}},{append:i(()=>[D(n(_,{icon:e(G)},null,8,["icon"]),[[P,a.value]])]),_:1},8,["modelValue"]),o("div",Y,[o("div",null,u(e(t)("tools.token.usage")),1),n(e(X),{modelValue:x,theme:f.value?"dark":"light",previewTheme:"github"},null,8,["theme"])])])):(m(),g("div",Z,[n(B,{title:e(t)("tools.token.tip.create"),icon:"info"},null,8,["title"])]))])]),_:1})])}}});export{re as default};
