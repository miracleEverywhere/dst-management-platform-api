import{t as e}from"./VRow-BASgO2lt.js";import{B as t,Bn as n,Hn as r,Ir as i,J as a,K as o,Lr as s,Qr as c,Rr as l,Tr as u,Un as d,Ur as f,Xr as p,Yr as m,Zn as h,_ as g,br as _,fi as v,li as y,lr as b,pi as x,ri as S,vr as C,wr as w,yr as T,z as E}from"./index-B5BjXw9j.js";import{t as D}from"./VAlert-DAD11G53.js";import{t as O}from"./preview-CGbxDVFx.js";import{t as k}from"./VSelect-D_EaPvwW.js";import{t as A}from"./VTextField-De8-o55X.js";import{t as j}from"./tools-gLRzfwHy.js";var M={class:`card-header`},N=g({__name:`token`,setup(g){let{t:N}=b(),P=n(),F=C(()=>P.theme),I=C(()=>h(P.language)),L=S({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:365*24},{title:N(`tools.token.select.permanent`),value:0}];S(!1);let z=S(``),B=()=>{if(L.value.expiration===void 0){r(N(`tools.token.noSelected`),`error`);return}j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,r(e.message,`success`)})},V=S(`\`\`\`python [id:Python]
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
\`\`\``),H=S(`\`\`\`golang [id:Golang]
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
\`\`\``),U=S(`\`\`\`java [id:Java]
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
\`\`\``),W=S("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=S(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=S(window.innerHeight),J=d(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return i(async()=>{window.addEventListener(`resize`,J)}),s(()=>{window.removeEventListener(`resize`,J)}),(n,r)=>{let i=f(`copy`);return l(),_(E,{height:Y()},{default:m(()=>[u(o,null,{default:m(()=>[T(`div`,M,[T(`span`,null,x(y(N)(`tools.token.title`)),1)])]),_:1}),u(t,{class:`mx-2`},{default:m(()=>[u(e,{class:`mt-4`},{default:m(()=>[u(D,{color:`warning`,density:`compact`},{default:m(()=>[w(x(y(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),y(z)===``?(l(),_(e,{key:0,class:`mt-8 d-flex align-center`},{default:m(()=>[u(k,{modelValue:y(L).expiration,"onUpdate:modelValue":r[0]||=e=>y(L).expiration=e,label:y(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),u(a,{size:`large`,class:`ml-4`,onClick:B},{default:m(()=>[w(x(y(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(l(),_(e,{key:1,class:`mt-8`},{default:m(()=>[u(A,{modelValue:y(z),"onUpdate:modelValue":r[1]||=e=>c(z)?z.value=e:null},{"append-inner":m(()=>[p(u(a,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[i,y(z)]])]),_:1},8,[`modelValue`])]),_:1})),u(e,{class:`mt-8`},{default:m(()=>[u(y(O),{"model-value":K,theme:y(F),language:y(I),"preview-theme":`github`,class:`mdp`,style:v({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-37c84146`]]);export{N as default};