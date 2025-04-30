import{t as B}from"./index-BAjU4gOB.js";import{u as P,a as V,g as E,c as v,o as I,r,J as l,K as z,e as _,f as k,j as n,k as i,l as o,t as u,i as e,p as M,n as j,ar as D,q as G,a0 as $,C as H,A as N}from"./index-C-KsYsfv.js";import{M as X}from"./preview-2M6rphTH.js";const J={class:"page-div"},O={class:"card-header"},F={style:{display:"flex"}},K={key:0},Q={class:"tip custom-block"},W={style:{"margin-top":"5px"}},Y={style:{"margin-top":"20px"}},Z=$({name:"toolsToken"}),re=Object.assign(Z,{setup(ee){const{t}=P();V();const m=E(),y=v(()=>m.language),f=v(()=>m.isDark);I(async()=>{});const s=r({expiredTime:null}),a=r(""),S=()=>{if(s.value.expiredTime===null){H(y.value==="zh"?"请选择过期时间":"Please select expire time");return}B.token.create.post(s.value).then(c=>{a.value=c.data,N(c.message)})},R=r(`\`\`\`python [id:Python]
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
\`\`\``),b=r(`\`\`\`golang [id:Golang]
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
\`\`\``),w=r(`\`\`\`java [id:Java]
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
\`\`\``),C=r("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'Authorization: token' \\\n--header 'X-I18n-Lang: lang'\n```"),q=r(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("Authorization", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-Json
\`\`\``),L=R.value+`

`+b.value+`

`+w.value+`

`+C.value+`

`+q.value;return(c,p)=>{const d=l("el-option"),x=l("el-select"),g=l("el-button"),T=l("el-input"),U=l("el-card"),A=z("copy");return _(),k("div",J,[n(U,{shadow:"never",style:{"min-height":"80vh"}},{header:i(()=>[o("div",O,[o("span",null,u(e(t)("tools.token.title")),1),o("div",F,[n(x,{modelValue:s.value.expiredTime,"onUpdate:modelValue":p[0]||(p[0]=h=>s.value.expiredTime=h),placeholder:e(t)("tools.token.expiredTime"),style:{width:"20vw","margin-right":"20px","font-weight":"lighter"}},{default:i(()=>[n(d,{label:e(t)("tools.token.options.day"),value:24},null,8,["label"]),n(d,{label:e(t)("tools.token.options.month"),value:720},null,8,["label"]),n(d,{label:e(t)("tools.token.options.year"),value:8760},null,8,["label"]),n(d,{label:e(t)("tools.token.options.forever"),value:8751240},null,8,["label"])]),_:1},8,["modelValue","placeholder"]),n(g,{type:"primary",onClick:S},{default:i(()=>[M(u(e(t)("tools.token.createButton")),1)]),_:1})])])]),default:i(()=>[o("div",null,[a.value?(_(),k("div",K,[o("div",Q,[o("div",W,u(e(t)("tools.token.tip.tip3")),1)]),n(T,{modelValue:a.value,"onUpdate:modelValue":p[1]||(p[1]=h=>a.value=h),style:{"max-width":"100%"}},{append:i(()=>[j(n(g,{icon:e(D)},null,8,["icon"]),[[A,a.value]])]),_:1},8,["modelValue"]),o("div",Y,[o("div",null,u(e(t)("tools.token.usage")),1),n(e(X),{modelValue:L,theme:f.value?"dark":"light",previewTheme:"github"},null,8,["theme"])])])):G("",!0)])]),_:1})])}}});export{re as default};
