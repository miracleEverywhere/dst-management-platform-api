import{u as B,a as U,g as A,c as P,o as V,r,B as d,C as D,d as g,e as v,f as a,w as p,i as e,t,j as n,m,aq as I,l as M,ar as z,n as E,X as j,Z as G,y as $}from"./index-iCREqSRD.js";import{M as H}from"./preview-Bq9SD73k.js";const N={class:"page-div"},X={class:"card-header"},Y={style:{display:"flex","align-items":"center"}},J={key:0},O={class:"tip custom-block"},F={style:{"font-weight":"bolder"}},Z={style:{"margin-top":"5px"}},K={style:{"margin-top":"20px"}},Q=j({name:"toolsToken"}),te=Object.assign(Q,{setup(W){const{t:o}=B();U();const _=A(),y=P(()=>_.isDark);V(async()=>{i.value.expiredTime=new Date().getTime()});const i=r({expiredTime:0}),s=r(""),k=()=>{G.token.create.post(i.value).then(u=>{s.value=u.data,$(u.message)})},f=r(`\`\`\`python [id:Python]
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
\`\`\``),S=r(`\`\`\`golang [id:Golang]
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
\`\`\``),R=r(`\`\`\`java [id:Java]
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
\`\`\``),w=r("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'Authorization: token' \\\n--header 'X-I18n-Lang: lang'\n```"),C=r(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("Authorization", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-Json
\`\`\``),x=f.value+`

`+S.value+`

`+R.value+`

`+w.value+`

`+C.value;return(u,l)=>{const h=d("el-button"),q=d("el-date-picker"),L=d("el-input"),T=d("el-card"),b=D("copy");return g(),v("div",N,[a(T,{shadow:"never",style:{"min-height":"80vh"}},{header:p(()=>[e("div",X,[e("span",null,t(n(o)("tools.token.title")),1),a(h,{type:"primary",onClick:k},{default:p(()=>[m(t(n(o)("tools.token.createButton")),1)]),_:1})])]),default:p(()=>[e("div",null,[e("div",Y,[e("span",null,t(n(o)("tools.token.expiredTime")),1),a(q,{modelValue:i.value.expiredTime,"onUpdate:modelValue":l[0]||(l[0]=c=>i.value.expiredTime=c),format:"YYYY-MM-DD",size:"large",style:{width:"160px","margin-left":"5px"},type:"date","value-format":"x"},null,8,["modelValue"])]),s.value?(g(),v("div",J,[e("div",O,[e("div",null,[m(t(n(o)("tools.token.tip.tip1"))+" ",1),e("span",F,t(n(I)(i.value.expiredTime)),1),m(" "+t(n(o)("tools.token.tip.tip2")),1)]),e("div",Z,t(n(o)("tools.token.tip.tip3")),1)]),a(L,{modelValue:s.value,"onUpdate:modelValue":l[1]||(l[1]=c=>s.value=c),style:{"max-width":"100%"}},{append:p(()=>[M(a(h,{icon:n(z)},null,8,["icon"]),[[b,s.value]])]),_:1},8,["modelValue"]),e("div",K,[e("div",null,t(n(o)("tools.token.usage")),1),a(n(H),{modelValue:x,theme:y.value?"dark":"light",previewTheme:"github"},null,8,["theme"])])])):E("",!0)])]),_:1})])}}});export{te as default};
