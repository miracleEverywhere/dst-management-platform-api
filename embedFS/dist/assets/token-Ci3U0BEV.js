import{Ar as e,Br as t,D as n,E as r,Fr as i,Gn as a,In as o,Jr as s,Ln as c,Nt as l,Or as u,T as d,Ur as f,V as p,Vr as m,_r as h,ei as g,fr as _,g as v,ii as y,kr as b,mr as x,ot as S,pr as C,ri as w,vr as T}from"./index-COBBxg0y.js";import{t as E}from"./VRow-CJWoaj-x.js";import{t as D}from"./VAlert-YljvnkN2.js";import{t as O}from"./preview-DoomFCum.js";import{t as k}from"./VSelect-CDI9-JhY.js";import{t as A}from"./VTextField-BgYX0-RR.js";import{t as j}from"./tools-mHKxON4M.js";var M={class:`card-header`},N=d({__name:`token`,setup(d){let{t:N}=n(),P=r(),F=_(()=>P.theme),I=_(()=>a(P.language)),L=s({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:365*24},{title:N(`tools.token.select.permanent`),value:0}];s(!1);let z=s(``),B=()=>{if(L.value.expiration===void 0){o(N(`tools.token.noSelected`),`error`);return}j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,o(e.message,`success`)})},V=s(`\`\`\`python [id:Python]
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
\`\`\``),H=s(`\`\`\`golang [id:Golang]
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
\`\`\``),U=s(`\`\`\`java [id:Java]
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
\`\`\``),W=s("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=s(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=s(window.innerHeight),J=c(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return u(async()=>{window.addEventListener(`resize`,J)}),b(()=>{window.removeEventListener(`resize`,J)}),(n,r)=>{let a=i(`copy`);return e(),x(p,{height:Y()},{default:t(()=>[T(l,null,{default:t(()=>[C(`div`,M,[C(`span`,null,y(g(N)(`tools.token.title`)),1)])]),_:1}),T(S,{class:`mx-2`},{default:t(()=>[T(E,{class:`mt-4`},{default:t(()=>[T(D,{color:`warning`,density:`compact`},{default:t(()=>[h(y(g(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),g(z)===``?(e(),x(E,{key:0,class:`mt-8 d-flex align-center`},{default:t(()=>[T(k,{modelValue:g(L).expiration,"onUpdate:modelValue":r[0]||=e=>g(L).expiration=e,label:g(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),T(v,{size:`large`,class:`ml-4`,onClick:B},{default:t(()=>[h(y(g(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(e(),x(E,{key:1,class:`mt-8`},{default:t(()=>[T(A,{modelValue:g(z),"onUpdate:modelValue":r[1]||=e=>f(z)?z.value=e:null},{"append-inner":t(()=>[m(T(v,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[a,g(z)]])]),_:1},8,[`modelValue`])]),_:1})),T(E,{class:`mt-8`},{default:t(()=>[T(g(O),{"model-value":K,theme:g(F),language:g(I),"preview-theme":`github`,class:`mdp`,style:w({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-37c84146`]]);export{N as default};