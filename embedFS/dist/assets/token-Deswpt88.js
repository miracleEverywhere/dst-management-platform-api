import{t as e}from"./VRow-DRTnW58O.js";import{B as t,Bn as n,Cr as r,Fr as i,Hn as a,Ir as o,J as s,K as c,Kr as l,Pr as u,Un as d,Vr as f,Yr as p,Zn as m,_ as h,br as g,ei as _,li as v,lr as y,oi as b,qr as x,ui as S,vr as C,wr as w,yr as T,z as E}from"./index-DkWZ5VPq.js";import{t as D}from"./VAlert-RxyCQr7q.js";import{t as O}from"./preview-gvwMMYKk.js";import{t as k}from"./VSelect-D57m4Fmi.js";import{t as A}from"./VTextField-BULm1O_S.js";import{t as j}from"./tools-Cty3CBC6.js";var M={class:`card-header`},N=h({__name:`token`,setup(h){let{t:N}=y(),P=n(),F=C(()=>P.theme),I=C(()=>m(P.language)),L=_({expiration:void 0}),R=[{title:N(`tools.token.select.day`),value:24},{title:N(`tools.token.select.week`),value:168},{title:N(`tools.token.select.month`),value:720},{title:N(`tools.token.select.year`),value:365*24},{title:N(`tools.token.select.permanent`),value:0}];_(!1);let z=_(``),B=()=>{if(L.value.expiration===void 0){a(N(`tools.token.noSelected`),`error`);return}j.token.post(L.value).then(e=>{z.value=e.data,L.value.expiration=void 0,a(e.message,`success`)})},V=_(`\`\`\`python [id:Python]
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
\`\`\``),H=_(`\`\`\`golang [id:Golang]
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
\`\`\``),U=_(`\`\`\`java [id:Java]
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
\`\`\``),W=_("```bash [id:cURL]\ncurl --location --globoff 'http://{ip}:{port}' \\\n--header 'X-DMP-TOKEN: token' \\\n--header 'X-I18n-Lang: lang'\n```"),G=_(`\`\`\`powershell [id:PowerShell]
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add("X-DMP-TOKEN", "token")
$headers.Add("X-I18n-Lang", "lang")

$response = Invoke-RestMethod 'http://{ip}:{port}' -Method 'GET' -Headers $headers
$response | ConvertTo-JSON
\`\`\``),K=V.value+`

`+H.value+`

`+U.value+`

`+W.value+`

`+G.value,q=_(window.innerHeight),J=d(()=>{q.value=window.innerHeight},200),Y=()=>Math.max(2,Math.floor(q.value-150));return u(async()=>{window.addEventListener(`resize`,J)}),i(()=>{window.removeEventListener(`resize`,J)}),(n,i)=>{let a=f(`copy`);return o(),g(E,{height:Y()},{default:l(()=>[w(c,null,{default:l(()=>[T(`div`,M,[T(`span`,null,S(b(N)(`tools.token.title`)),1)])]),_:1}),w(t,{class:`mx-2`},{default:l(()=>[w(e,{class:`mt-4`},{default:l(()=>[w(D,{color:`warning`,density:`compact`},{default:l(()=>[r(S(b(N)(`tools.token.tip`)),1)]),_:1})]),_:1}),b(z)===``?(o(),g(e,{key:0,class:`mt-8 d-flex align-center`},{default:l(()=>[w(k,{modelValue:b(L).expiration,"onUpdate:modelValue":i[0]||=e=>b(L).expiration=e,label:b(N)(`tools.token.select.label`),items:R},null,8,[`modelValue`,`label`]),w(s,{size:`large`,class:`ml-4`,onClick:B},{default:l(()=>[r(S(b(N)(`tools.token.create`)),1)]),_:1})]),_:1})):(o(),g(e,{key:1,class:`mt-8`},{default:l(()=>[w(A,{modelValue:b(z),"onUpdate:modelValue":i[1]||=e=>p(z)?z.value=e:null},{"append-inner":l(()=>[x(w(s,{variant:`text`,icon:`ri-file-copy-line`},null,512),[[a,b(z)]])]),_:1},8,[`modelValue`])]),_:1})),w(e,{class:`mt-8`},{default:l(()=>[w(b(O),{"model-value":K,theme:b(F),language:b(I),"preview-theme":`github`,class:`mdp`,style:v({"overflow-y":`auto`,height:Y()-220+`px`})},null,8,[`theme`,`language`,`style`])]),_:1})]),_:1})]),_:1},8,[`height`])}}},[[`__scopeId`,`data-v-37c84146`]]);export{N as default};