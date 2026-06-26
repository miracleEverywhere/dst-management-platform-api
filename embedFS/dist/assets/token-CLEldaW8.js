import{$r as e,Br as t,D as n,Dr as r,E as i,Gn as a,Hr as o,In as s,Ln as c,Nt as l,Or as u,Pr as d,T as f,V as p,_r as m,dr as h,fr as g,g as _,gr as v,kr as y,ni as b,ot as x,pr as S,qr as C,ri as w,zr as T}from"./index-dmWdwIy7.js";import{t as E}from"./VRow-BxogcfR0.js";import{t as D}from"./VAlert-JLXgyHbW.js";import{t as O}from"./preview-Ch1ih2lH.js";import{t as k}from"./VSelect-DS94psfl.js";import{t as A}from"./VTextField-DkWN2IrB.js";import{t as j}from"./tools-C_qak2Yy.js";var M={class:`card-header`},N=f({__name:`token`,setup(f){let{t:N}=n(),P=i(),F=h(()=>P.theme),I=h(()=>a(P.language)),L=C({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:365*24},{title:N(`tools.token.select.permanent`),value:0}];C(!1);let z=C(``),B=()=>{if(L.value.expiration===void 0){s(N(`tools.token.noSelected`),`error`);return}j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,s(e.message,`success`)})},V=C(`\`\`\`python [id:Python]
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
\`\`\``),H=C(`\`\`\`golang [id:Golang]
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
\`\`\``),U=C(`\`\`\`java [id:Java]
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
\`\`\``),W=C("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=C(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=C(window.innerHeight),J=c(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return r(async()=>{window.addEventListener(`resize`,J)}),u(()=>{window.removeEventListener(`resize`,J)}),(n,r)=>{let i=d(`copy`);return y(),S(p,{height:Y()},{default:T(()=>[m(l,null,{default:T(()=>[g(`div`,M,[g(`span`,null,w(e(N)(`tools.token.title`)),1)])]),_:1}),m(x,{class:`mx-2`},{default:T(()=>[m(E,{class:`mt-4`},{default:T(()=>[m(D,{color:`warning`,density:`compact`},{default:T(()=>[v(w(e(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),e(z)===``?(y(),S(E,{key:0,class:`mt-8 d-flex align-center`},{default:T(()=>[m(k,{modelValue:e(L).expiration,"onUpdate:modelValue":r[0]||=t=>e(L).expiration=t,label:e(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),m(_,{size:`large`,class:`ml-4`,onClick:B},{default:T(()=>[v(w(e(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(y(),S(E,{key:1,class:`mt-8`},{default:T(()=>[m(A,{modelValue:e(z),"onUpdate:modelValue":r[1]||=e=>o(z)?z.value=e:null},{"append-inner":T(()=>[t(m(_,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[i,e(z)]])]),_:1},8,[`modelValue`])]),_:1})),m(E,{class:`mt-8`},{default:T(()=>[m(e(O),{"model-value":K,theme:e(F),language:e(I),"preview-theme":`github`,class:`mdp`,style:b({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-37c84146`]]);export{N as default};