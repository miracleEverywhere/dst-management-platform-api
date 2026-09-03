import{t as e}from"./VRow-BwUQqqAr.js";import{$r as t,B as n,Bn as r,Er as i,Hn as a,J as o,K as s,Lr as c,Rr as l,Tr as u,Un as d,Wr as f,Xr as p,Zn as m,Zr as h,_ as g,br as _,ii as v,mi as y,pi as b,ui as x,ur as S,xr as C,yr as w,z as T,zr as E}from"./index-D3EtMxce.js";import{t as D}from"./VAlert-7CQ0FZ6v.js";import{t as O}from"./preview-D1yvVSXa.js";import{t as k}from"./VSelect-BTKT_8op.js";import{t as A}from"./VTextField-BLsFFFgZ.js";import{t as j}from"./tools-BL5FM4Rl.js";var M={class:`card-header`},N=g({__name:`token`,setup(g){let{t:N}=S(),P=r(),F=w(()=>P.theme),I=w(()=>m(P.language)),L=v({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:365*24},{title:N(`tools.token.select.permanent`),value:0}];v(!1);let z=v(``),B=()=>{if(L.value.expiration===void 0){a(N(`tools.token.noSelected`),`error`);return}j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,a(e.message,`success`)})},V=v(`\`\`\`python [id:Python]
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
\`\`\``),H=v(`\`\`\`golang [id:Golang]
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
\`\`\``),U=v(`\`\`\`java [id:Java]
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
\`\`\``),W=v("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=v(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=v(window.innerHeight),J=d(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return c(async()=>{window.addEventListener(`resize`,J)}),l(()=>{window.removeEventListener(`resize`,J)}),(r,a)=>{let c=f(`copy`);return E(),C(T,{height:Y()},{default:p(()=>[i(s,null,{default:p(()=>[_(`div`,M,[_(`span`,null,y(x(N)(`tools.token.title`)),1)])]),_:1}),i(n,{class:`mx-2`},{default:p(()=>[i(e,{class:`mt-4`},{default:p(()=>[i(D,{color:`warning`,density:`compact`},{default:p(()=>[u(y(x(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),x(z)===``?(E(),C(e,{key:0,class:`mt-8 d-flex align-center`},{default:p(()=>[i(k,{modelValue:x(L).expiration,"onUpdate:modelValue":a[0]||=e=>x(L).expiration=e,label:x(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),i(o,{size:`large`,class:`ml-4`,onClick:B},{default:p(()=>[u(y(x(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(E(),C(e,{key:1,class:`mt-8`},{default:p(()=>[i(A,{modelValue:x(z),"onUpdate:modelValue":a[1]||=e=>t(z)?z.value=e:null},{"append-inner":p(()=>[h(i(o,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[c,x(z)]])]),_:1},8,[`modelValue`])]),_:1})),i(e,{class:`mt-8`},{default:p(()=>[i(x(O),{"model-value":K,theme:x(F),language:x(I),"preview-theme":`github`,class:`mdp`,style:b({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-37c84146`]]);export{N as default};