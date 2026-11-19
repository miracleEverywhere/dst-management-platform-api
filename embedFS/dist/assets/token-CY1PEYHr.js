import{n as e,u as t}from"./tools-DVD_XdJz.js";import{I as n,Mt as r,N as i,P as a,Q as o,V as s,Z as c,_ as l,d as u,jt as d,l as f,mt as p,nt as m,st as h,u as g,v as _}from"./runtime-core.esm-bundler-CaNi8Aqu.js";import{t as v}from"./snackbar-51_Gua36.js";import{t as y}from"./global-DcB-jhtp.js";import{t as b}from"./VRow-Cz-G3xvP.js";import{t as x}from"./preview-CkfmoI8H.js";import{D as S,I as C,L as w,O as T,V as E,X as D,z as O}from"./index-BHqUloHm.js";import{t as k}from"./VAlert-CccK434J.js";import{t as A}from"./VSelect-B8181i23.js";import{t as j}from"./VTextField-C13zjSMT.js";import{t as M}from"./tools-DrIeAq5h.js";var N={class:`card-header`},P=S({__name:`token`,setup(S){let{t:P}=D(),F=y(),I=f(()=>T(F.theme)),L=f(()=>t(F.language)),R=h({expiration:void 0}),z=[{title:P(`tools.token.select.day`),value:24},{title:P(`tools.token.select.week`),value:168},{title:P(`tools.token.select.month`),value:720},{title:P(`tools.token.select.year`),value:8760},{title:P(`tools.token.select.permanent`),value:0}];h(!1);let B=h(``),V=()=>{R.value.expiration===void 0?v(P(`tools.token.noSelected`),`error`):M.token.post(R.value).then(e=>{B.value=e.data,R.value.expiration=void 0,v(e.message,`success`)})},H=h(`\`\`\`python [id:Python]
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
\`\`\``),U=h(`\`\`\`golang [id:Golang]
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
\`\`\``),W=h(`\`\`\`java [id:Java]
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
\`\`\``),G=h("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),K=h(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),q=H.value+`

`+U.value+`

`+W.value+`

`+G.value+`

`+K.value,J=h(window.innerHeight),Y=e(()=>{J.value=window.innerHeight},200),X=()=>Math.max(2,Math.floor(J.value-150));return i(async()=>{window.addEventListener(`resize`,Y)}),a(()=>{window.removeEventListener(`resize`,Y)}),(e,t)=>{let i=s(`copy`);return n(),u(C,{height:X()},{default:c(()=>[_(O,null,{default:c(()=>[g(`div`,N,[g(`span`,null,r(p(P)(`tools.token.title`)),1)])]),_:1}),_(w,{class:`mx-2`},{default:c(()=>[_(b,{class:`mt-4`},{default:c(()=>[_(k,{color:`warning`,density:`compact`},{default:c(()=>[l(r(p(P)(`tools.token.tip`)),1)]),_:1})]),_:1}),p(B)===``?(n(),u(b,{key:0,class:`mt-8 d-flex align-center`},{default:c(()=>[_(A,{modelValue:p(R).expiration,"onUpdate:modelValue":t[0]||=e=>p(R).expiration=e,label:p(P)(`tools.token.select.label`),items:z},null,8,[`modelValue`,`label`]),_(E,{size:`large`,class:`ml-4`,onClick:V},{default:c(()=>[l(r(p(P)(`tools.token.create`)),1)]),_:1})]),_:1})):(n(),u(b,{key:1,class:`mt-8`},{default:c(()=>[_(j,{modelValue:p(B),"onUpdate:modelValue":t[1]||=e=>m(B)?B.value=e:null},{"append-inner":c(()=>[o(_(E,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[i,p(B)]])]),_:1},8,[`modelValue`])]),_:1})),_(b,{class:`mt-8`},{default:c(()=>[_(p(x),{"model-value":q,theme:p(I),language:p(L),"preview-theme":`github`,class:`mdp`,style:d({"overflow-y":`auto`,height:X()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-a5088aba`]]);export{P as default};