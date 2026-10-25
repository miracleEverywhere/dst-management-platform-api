import{n as e,u as t}from"./tools-BDiA29KN.js";import{I as n,Mt as r,N as i,Nt as a,P as o,Q as s,V as c,Z as l,_ as u,ct as d,d as f,ht as p,l as m,rt as h,u as g,v as _}from"./runtime-core.esm-bundler-C1CWKYjf.js";import{t as v}from"./snackbar-Cc_fot4v.js";import{t as y}from"./global-CL_Am_h6.js";import{t as b}from"./VRow-CtlwYQ50.js";import{t as x}from"./preview-CGgBvuej.js";import{D as S,F as C,L as w,P as T,q as E,z as D}from"./index-4lttKsEn.js";import{t as O}from"./VAlert-DuHE7WSs.js";import{t as k}from"./VSelect-DxP39aRE.js";import{t as A}from"./VTextField-BwVFAZZh.js";import{t as j}from"./tools-DRasd9Ej.js";var M={class:`card-header`},N=S({__name:`token`,setup(S){let{t:N}=E(),P=y(),F=m(()=>P.theme),I=m(()=>t(P.language)),L=d({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:8760},{title:N(`tools.token.select.permanent`),value:0}];d(!1);let z=d(``),B=()=>{L.value.expiration===void 0?v(N(`tools.token.noSelected`),`error`):j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,v(e.message,`success`)})},V=d(`\`\`\`python [id:Python]
# pip install dmp-sdk-python (安装python-sdk)
from dmp_sdk_python import DMPClient

# 初始化客户端（通过 token 认证）
client = DMPClient("http://your-server:80", "your-token")


# 链式调用: client.模块.方法()
users = client.user.list_users()
print(users.rows)

rooms = client.room.list()
print(rooms.rows)

room_info = client.rm.get(room_id=8)
print(room_info)

mods = client.mod.get_enabled(roomID=8, worldID=24)
print(mods)

sys_info = client.pt.os_info()
print(sys_info)

cpu_usage = client.dashboard.get_sys_info()['cpu']
print(cpu_usage)
\`\`\``),H=d(`\`\`\`golang [id:Golang]
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
\`\`\``),U=d(`\`\`\`java [id:Java]
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
\`\`\``),W=d("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=d(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=d(window.innerHeight),J=e(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return i(async()=>{window.addEventListener(`resize`,J)}),o(()=>{window.removeEventListener(`resize`,J)}),(e,t)=>{let i=c(`copy`);return n(),f(T,{height:Y()},{default:l(()=>[_(w,null,{default:l(()=>[g(`div`,M,[g(`span`,null,a(p(N)(`tools.token.title`)),1)])]),_:1}),_(C,{class:`mx-2`},{default:l(()=>[_(b,{class:`mt-4`},{default:l(()=>[_(O,{color:`warning`,density:`compact`},{default:l(()=>[u(a(p(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),p(z)===``?(n(),f(b,{key:0,class:`mt-8 d-flex align-center`},{default:l(()=>[_(k,{modelValue:p(L).expiration,"onUpdate:modelValue":t[0]||=e=>p(L).expiration=e,label:p(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),_(D,{size:`large`,class:`ml-4`,onClick:B},{default:l(()=>[u(a(p(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(n(),f(b,{key:1,class:`mt-8`},{default:l(()=>[_(A,{modelValue:p(z),"onUpdate:modelValue":t[1]||=e=>h(z)?z.value=e:null},{"append-inner":l(()=>[s(_(D,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[i,p(z)]])]),_:1},8,[`modelValue`])]),_:1})),_(b,{class:`mt-8`},{default:l(()=>[_(p(x),{"model-value":K,theme:p(F),language:p(I),"preview-theme":`github`,class:`mdp`,style:r({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-da692352`]]);export{N as default};