import{u as ee,a as te,g as ne,c as le,r as u,K as p,f as s,i as r,k as l,l as o,m as i,q as h,t as n,j as e,n as E,v as g,a1 as oe,a3 as ie,B as se}from"./index-4XnH3GwD.js";import{l as ae}from"./index-18hcdDlX.js";import{t as de}from"./index-pDK93l1X.js";import{M as R}from"./preview-C7Lzfx8s.js";const re={class:"page-div"},_e={style:{margin:"20px"}},pe={style:{"margin-bottom":"40px"}},he={style:{"font-weight":"bolder","font-size":"16px"}},ue={key:1,style:{"font-weight":"bolder","font-size":"16px"}},ce={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},me={style:{"line-height":"50px"}},fe={style:{"line-height":"50px"}},ge={class:"tip custom-block"},be={class:"custom-block-title"},ve={style:{width:"50vh",height:"70vh"}},ye={style:{"font-weight":"bolder","font-size":"16px"}},we={key:1,style:{"font-weight":"bolder","font-size":"16px"}},xe={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},Ee={class:"tip_success custom-block"},ke={class:"custom-block-title"},Ae={class:"tip custom-block"},Te={class:"custom-block-title",style:{"text-decoration":"line-through"}},Se={style:{"line-height":"50px","text-decoration":"line-through"}},Re={style:{"line-height":"50px","text-decoration":"line-through"}},Oe={style:{"line-height":"50px","font-weight":"bolder","font-size":"14px"}},Ve={style:{"font-weight":"bolder","font-size":"16px"}},Le={key:1,style:{"font-weight":"bolder","font-size":"16px"}},Ne={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},Ce={style:{"line-height":"50px"}},De={style:{"line-height":"50px"}},Me={style:{"line-height":"50px"}},Ie={style:{"line-height":"50px"}},Fe={style:{"font-weight":"bolder","font-size":"16px"}},Be={key:1,style:{"font-weight":"bolder","font-size":"16px"}},Ue={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},ze={style:{"line-height":"50px"}},Ge={style:{"line-height":"50px"}},Pe={style:{"font-weight":"bolder","font-size":"16px"}},Ye={key:1,style:{"font-weight":"bolder","font-size":"16px"}},He={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},Ke={class:"tip custom-block"},Je={class:"custom-block-title"},Ze={class:"tip custom-block"},We={class:"custom-block-title"},je={style:{"font-weight":"bolder","font-size":"16px"}},qe={key:1,style:{"font-weight":"bolder","font-size":"16px"}},Xe={key:0,style:{"line-height":"50px","font-weight":"bold",color:"#409EFF"}},$e={style:{"line-height":"50px","font-weight":"bolder"}},Qe={style:{"line-height":"50px","font-weight":"bolder"}},et={style:{"line-height":"50px","font-weight":"bolder"}},tt={class:"tip"},nt={style:{"line-height":"50px","font-weight":"bolder"}},lt={key:1},ot={key:2},it={style:{"line-height":"50px","font-weight":"bolder"}},st={key:3},at={key:4},dt=oe({name:"help"}),ct=Object.assign(dt,{setup(rt){const{t}=ee(),{isMobile:T}=te(),O=ne(),y=le(()=>O.isDark),V=u("0"),L=new URL("/assets/1-CGTISVuJ.JPG",import.meta.url).href,N=new URL("/assets/master-light-GRmZlU9Y.png",import.meta.url).href,C=new URL("/assets/slave-light-CZ_JIs4D.png",import.meta.url).href,D=new URL("/assets/master-dark-Cz2T4TCk.png",import.meta.url).href,M=new URL("/assets/slave-dark-C983RHxS.png",import.meta.url).href,F=u(),B=u(`\`\`\`shell ::close
# 备份
cd ~
mv dst/bin/lib32/steamclient.so dst/bin/lib32/steamclient.so.bak
mv dst/steamclient.so dst/steamclient.so.bak
# 替换
cp steamcmd/linux32/steamclient.so dst/bin/lib32/steamclient.so
cp steamcmd/linux32/steamclient.so dst/steamclient.so
\`\`\``),U=u(),z=u(`\`\`\`lua ::open
return {
  background_node_range={ 0, 1 },
  desc="你敢去熔炉里证明你自己的实力吗？",
  hideminimap=false,
  id="LAVAARENA",
  location="lavaarena",
  max_playlist_position=999,
  min_playlist_position=0,
  name="熔炉",
  numrandom_set_pieces=0,
  override_level_string=false,
  overrides={
    autumn="default",
    basicresource_regrowth="none",
    beefaloheat="default",
    boons="never",
    brightmarecreatures="default",
    crow_carnival="default",
    darkness="default",
    day="default",
    dropeverythingondespawn="default",
    extrastartingitems="default",
    ghostenabled="always",
    ghostsanitydrain="always",
    hallowed_nights="default",
    healthpenalty="always",
    hunger="default",
    keep_disconnected_tiles=true,
    krampus="default",
    layout_mode="RestrictNodesByKey",
    lessdamagetaken="none",
    no_joining_islands=true,
    no_wormholes_to_disconnected_tiles=true,
    poi="never",
    portalresurection="none",
    protected="never",
    resettime="default",
    roads="never",
    season_start="default",
    seasonalstartingitems="default",
    shadowcreatures="default",
    spawnmode="fixed",
    spawnprotection="default",
    specialevent="default",
    spring="default",
    start_location="lavaarena",
    summer="default",
    task_set="lavaarena_taskset",
    temperaturedamage="default",
    touchstone="never",
    traps="never",
    winter="default",
    winters_feast="default",
    world_size="small",
    year_of_the_beefalo="default",
    year_of_the_bunnyman="default",
    year_of_the_carrat="default",
    year_of_the_catcoon="default",
    year_of_the_dragonfly="default",
    year_of_the_gobbler="default",
    year_of_the_pig="default",
    year_of_the_varg="default"
  },
  required_prefabs={ "lavaarena_portal" },
  settings_desc="你敢去熔炉里证明你自己的实力吗？",
  settings_id="LAVAARENA",
  settings_name="熔炉",
  substitutes={  },
  version=2,
  worldgen_desc="你敢去熔炉里证明你自己的实力吗？",
  worldgen_id="LAVAARENA",
  worldgen_name="熔炉"
}
\`\`\``),G=u(),P=u(`\`\`\`lua ::open
return {
  ["workshop-1938752683"]={
    configuration_options={
      ADJUST_FILTER=false,
      BATTLESTANDARD_EFFICIENCY=1,
      COMMAND_SPAM_BAN_TIME=10,
      DAMAGE_NUMBER_FONT_SIZE=32,
      DAMAGE_NUMBER_HEIGHT=40,
      DAMAGE_NUMBER_OPTIONS="default",
      DAMAGE_NUMBER_PLAYERS=false,
      DEBUG=false,
      DEFAULT_FILTER=1,
      DEFAULT_LOBBY_TAB="news",
      DEFAULT_ROTATION=false,
      DIFFICULTY="normal",
      DISPLAY_COLORED_STATS=true,
      DISPLAY_TARGET_BADGE=true,
      DISPLAY_TEAMMATES_DEBUFFS=false,
      ["Damage Number Options"]=0,
      ["Detailed Summary Options"]=0,
      EVENT_TRACKING=true,
      FORCE_START_DELAY_TIME=5,
      FRIENDLY_FIRE=false,
      GAMETYPE="forge",
      GIFT_SIDE="right",
      ["Gameplay Settings"]=0,
      HIDE_INDICATORS=true,
      JOINABLE_MIDMATCH=true,
      LOBBY_GEAR=true,
      ["Lobby Options"]=0,
      MAX_MESSAGES=100,
      MOB_ATTACK_RATE=1,
      MOB_DAMAGE_DEALT=1,
      MOB_DAMAGE_TAKEN=1,
      MOB_DUPLICATOR=1,
      MOB_HEALTH=1,
      MOB_SIZE=1,
      MOB_SPEED=1,
      MODE="reforged",
      Mutators=0,
      NO_HUD=false,
      NO_REVIVES=false,
      NO_SLEEP=false,
      ONLY_SHOW_NONZERO_STATS=true,
      Other=0,
      PING_KEYBIND="KEY_R",
      PING_TRANSPARENCY=100,
      PLAYER_DEBUFF_DISPLAY="mini",
      ["Player HUD Options"]=0,
      RESERVE_SLOTS=true,
      ROTATION=0,
      SANDBOX=false,
      SERVER_ACHIEVEMENTS=false,
      SERVER_LEVEL=false,
      SHOW_CHAT_ICON=false,
      SPECTATORS_ONLY=true,
      SPECTATOR_ON_DEATH=false,
      VOTE_FORCE_START=true,
      VOTE_GAME_SETTINGS=true,
      VOTE_KICK=true,
      ["Visual Options"]=0,
      Vote=0,
      WAVESET="swineclops"
    },
    enabled=true
  },
  ["workshop-2038128735"]={ configuration_options={ klaustrophobia=false }, enabled=true },
  ["workshop-2619860122"]={ configuration_options={  }, enabled=true },
  ["workshop-2633870801"]={
    configuration_options={
      ["Gameplay Settings"]=0,
      MAP="none",
      ["Other Settings"]=0,
      WAVESET="none",
      light_color_override=false
    },
    enabled=true
  },
  ["workshop-2961923603"]={ configuration_options={  }, enabled=true },
  ["workshop-3132633883"]={ configuration_options={  }, enabled=true },
  ["workshop-3139080374"]={
    configuration_options={
      Brainwash_Fix=true,
      Lock_Recipes=true,
      Manually_Rapid_Atk=true,
      Random_Character_Fix=true,
      Rhinocebro_Fix=false,
      Spike_Fix=true,
      Tenfold_Optimize=true,
      force_camera=true,
      worly_cookpot=false
    },
    enabled=true
  },
  ["workshop-666155465"]={
    configuration_options={
      chestB=-1,
      chestG=-1,
      chestR=-1,
      display_hp=-1,
      food_estimation=-1,
      food_order=0,
      food_style=0,
      lang="auto",
      show_food_units=-1,
      show_uses=-1
    },
    enabled=true
  }
}
\`\`\``);u(),u("```ini ::open\n[ACCOUNT]\nencode_user_path = true\n```"),u(),u("```ini ::open\n[ACCOUNT]\nencode_user_path = false\n```");const Y=()=>{},S=u(!1),H=()=>{S.value=!0;const f={clusterName:O.selectedDstCluster};ae.download.post(f).then(async a=>{await ie(a.data,"logs.tgz")}).finally(()=>{S.value=!1})},K=()=>{de.replaceSo.post().then(f=>{se(f.message)})},c=f=>T.value&&f.length>25?f.substring(0,25)+"...":f,m=f=>T.value?f.length>25:!1,_=u({accessLog:"dmp.log",runtimeLog:"dmpProcess.log",backup:"dmp_files/backup/",uidMap:"dmp_files/uid_map/",mod:"dmp_files/mod/",config:"~/.klei/DoNotStarveTogether/",game:"dst/",steam:"steamcmd/"});return(f,a)=>{const J=p("el-text"),Z=p("el-link"),w=p("el-tooltip"),k=p("el-image"),x=p("el-collapse-item"),I=p("el-button"),A=p("el-timeline-item"),W=p("el-timeline"),b=p("el-input"),v=p("el-form-item"),j=p("el-form"),q=p("el-collapse"),X=p("el-card"),$=p("el-col"),Q=p("el-row");return s(),r("div",re,[l(Q,{gutter:10},{default:o(()=>[l($,{lg:24,md:24,sm:24,span:24,xs:24,style:{"margin-top":"10px"}},{default:o(()=>[l(X,{shadow:"never",style:{"min-height":"80vh"}},{default:o(()=>[i("div",_e,[i("div",pe,[l(J,{type:"info"},{default:o(()=>[h(n(e(t)("help.header.text1")),1)]),_:1}),l(Z,{type:"primary",underline:"never",href:"https://miraclesses.top",target:"_blank"},{default:o(()=>a[9]||(a[9]=[h(" https://miraclesses.top ")])),_:1})]),l(q,{modelValue:V.value,"onUpdate:modelValue":a[8]||(a[8]=d=>V.value=d),accordion:"",onChange:Y},{default:o(()=>[l(x,{name:"1"},{title:o(()=>[m(e(t)("help.one.title"))?(s(),E(w,{key:0,content:e(t)("help.one.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",he,n(c(e(t)("help.one.title"))),1)]),_:1},8,["content"])):(s(),r("span",ue,n(c(e(t)("help.one.title"))),1))]),default:o(()=>[m(e(t)("help.one.title"))?(s(),r("div",ce,n(e(t)("help.one.title")),1)):g("",!0),i("div",me,n(e(t)("help.one.text1")),1),i("div",fe,n(e(t)("help.one.text2")),1),i("div",ge,[i("p",be,n(e(t)("help.one.text3")),1)]),i("div",ve,[l(k,{"hide-on-click-modal":!0,"initial-index":4,"max-scale":7,"min-scale":.2,"preview-src-list":[e(L)],src:e(L),"zoom-rate":1.2,fit:"contain",style:{"margin-top":"10px","margin-bottom":"10px"}},null,8,["preview-src-list","src"])])]),_:1}),l(x,{name:"2"},{title:o(()=>[m(e(t)("help.two.title"))?(s(),E(w,{key:0,content:e(t)("help.two.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",ye,n(c(e(t)("help.two.title"))),1)]),_:1},8,["content"])):(s(),r("span",we,n(c(e(t)("help.two.title"))),1))]),default:o(()=>[m(e(t)("help.two.title"))?(s(),r("div",xe,n(e(t)("help.two.title")),1)):g("",!0),i("div",Ee,[i("span",ke,n(e(t)("help.two.text2_6")),1)]),i("div",Ae,[i("p",Te,n(e(t)("help.two.text2_2")),1),l(I,{type:"primary",disabled:"",onClick:K},{default:o(()=>[h(n(e(t)("help.two.button_1")),1)]),_:1})]),i("div",Se,[h(n(e(t)("help.two.text1"))+" ",1),i("code",null,n(e(t)("help.two.text1_1")),1),h(" "+n(e(t)("help.two.text1_2"))+" ",1),i("code",null,n(e(t)("help.two.text1_3")),1),h(" "+n(e(t)("help.two.text1_4")),1)]),l(e(R),{ref_key:"twoCodeRef",ref:F,modelValue:B.value,theme:y.value?"dark":"light",previewTheme:"github"},null,8,["modelValue","theme"]),i("div",Re,n(e(t)("help.two.text3")),1),i("div",Oe,n(e(t)("help.two.timeline")),1),l(W,{style:{"max-width":"600px"}},{default:o(()=>[l(A,{size:"large",timestamp:"2024-10-25",type:"primary"},{default:o(()=>[h(n(e(t)("help.two.text2_4")),1)]),_:1}),l(A,{size:"large",timestamp:"2024-11-7",type:"danger"},{default:o(()=>[h(n(e(t)("help.two.text2")),1)]),_:1}),l(A,{size:"large",timestamp:"2024-12-8",type:"danger"},{default:o(()=>[h(n(e(t)("help.two.text2")),1)]),_:1}),l(A,{size:"large",timestamp:"2024-12-9",type:"warning"},{default:o(()=>[h(n(e(t)("help.two.text2_3")),1)]),_:1}),l(A,{size:"large",timestamp:"2025-3-18",type:"warning"},{default:o(()=>[h(n(e(t)("help.two.text2_5")),1)]),_:1})]),_:1})]),_:1}),l(x,{name:"3"},{title:o(()=>[m(e(t)("help.three.title"))?(s(),E(w,{key:0,content:e(t)("help.three.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",Ve,n(c(e(t)("help.three.title"))),1)]),_:1},8,["content"])):(s(),r("span",Le,n(c(e(t)("help.three.title"))),1))]),default:o(()=>[m(e(t)("help.three.title"))?(s(),r("div",Ne,n(e(t)("help.three.title")),1)):g("",!0),i("div",Ce,n(e(t)("help.three.text1")),1),i("div",De,n(e(t)("help.three.text2")),1),l(e(R),{ref_key:"threeCodeOneRef",ref:U,modelValue:z.value,theme:y.value?"dark":"light",previewTheme:"github"},null,8,["modelValue","theme"]),i("div",Me,n(e(t)("help.three.text3")),1),i("div",Ie,n(e(t)("help.three.text4")),1),l(e(R),{ref_key:"threeCodeTwoRef",ref:G,modelValue:P.value,theme:y.value?"dark":"light",previewTheme:"github"},null,8,["modelValue","theme"])]),_:1}),l(x,{name:"4"},{title:o(()=>[m(e(t)("help.four.title"))?(s(),E(w,{key:0,content:e(t)("help.four.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",Fe,n(c(e(t)("help.four.title"))),1)]),_:1},8,["content"])):(s(),r("span",Be,n(c(e(t)("help.four.title"))),1))]),default:o(()=>[m(e(t)("help.four.title"))?(s(),r("div",Ue,n(e(t)("help.four.title")),1)):g("",!0),i("div",ze,n(e(t)("help.four.text1")),1),i("div",Ge,[h(n(e(t)("help.four.text2"))+" ",1),l(I,{loading:S.value,size:"small",type:"success",onClick:H},{default:o(()=>[h(n(e(t)("help.four.button")),1)]),_:1},8,["loading"]),h(" "+n(e(t)("help.four.text3")),1)])]),_:1}),l(x,{name:"8"},{title:o(()=>[m(e(t)("help.eight.title"))?(s(),E(w,{key:0,content:e(t)("help.eight.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",Pe,n(c(e(t)("help.eight.title"))),1)]),_:1},8,["content"])):(s(),r("span",Ye,n(c(e(t)("help.eight.title"))),1))]),default:o(()=>[m(e(t)("help.eight.title"))?(s(),r("div",He,n(e(t)("help.eight.title")),1)):g("",!0),l(j,{"label-width":"120","label-position":e(T)?"top":"left"},{default:o(()=>[i("div",Ke,[i("span",Je,n(e(t)("help.eight.tip1")),1)]),l(v,{label:e(t)("help.eight.dmp.accessLog")},{default:o(()=>[l(b,{modelValue:_.value.accessLog,"onUpdate:modelValue":a[0]||(a[0]=d=>_.value.accessLog=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:e(t)("help.eight.dmp.runtimeLog")},{default:o(()=>[l(b,{modelValue:_.value.runtimeLog,"onUpdate:modelValue":a[1]||(a[1]=d=>_.value.runtimeLog=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:e(t)("help.eight.dmp.backup")},{default:o(()=>[l(b,{modelValue:_.value.backup,"onUpdate:modelValue":a[2]||(a[2]=d=>_.value.backup=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:e(t)("help.eight.dmp.uidMap")},{default:o(()=>[l(b,{modelValue:_.value.uidMap,"onUpdate:modelValue":a[3]||(a[3]=d=>_.value.uidMap=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:e(t)("help.eight.dmp.mod")},{default:o(()=>[l(b,{modelValue:_.value.mod,"onUpdate:modelValue":a[4]||(a[4]=d=>_.value.mod=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),i("div",Ze,[i("span",We,n(e(t)("help.eight.tip2")),1)]),l(v,{label:e(t)("help.eight.dst.config")},{default:o(()=>[l(b,{modelValue:_.value.config,"onUpdate:modelValue":a[5]||(a[5]=d=>_.value.config=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:e(t)("help.eight.dst.game")},{default:o(()=>[l(b,{modelValue:_.value.game,"onUpdate:modelValue":a[6]||(a[6]=d=>_.value.game=d),disabled:""},null,8,["modelValue"])]),_:1},8,["label"]),l(v,{label:"Steam"},{default:o(()=>[l(b,{modelValue:_.value.steam,"onUpdate:modelValue":a[7]||(a[7]=d=>_.value.steam=d),disabled:""},null,8,["modelValue"])]),_:1})]),_:1},8,["label-position"])]),_:1}),l(x,{name:"9"},{title:o(()=>[m(e(t)("help.nine.title"))?(s(),E(w,{key:0,content:e(t)("help.nine.title"),effect:"light",placement:"top"},{default:o(()=>[i("span",je,n(c(e(t)("help.nine.title"))),1)]),_:1},8,["content"])):(s(),r("span",qe,n(c(e(t)("help.nine.title"))),1))]),default:o(()=>[m(e(t)("help.nine.title"))?(s(),r("div",Xe,n(e(t)("help.nine.title")),1)):g("",!0),i("div",$e,n(e(t)("help.nine.text1")),1),i("div",Qe,n(e(t)("help.nine.text2")),1),i("div",et,n(e(t)("help.nine.text3")),1),i("div",tt,n(e(t)("help.nine.tip1")),1),i("div",nt,n(e(t)("help.nine.text4")),1),y.value?g("",!0):(s(),r("div",lt,[l(k,{"hide-on-click-modal":!0,"initial-index":4,"max-scale":7,"min-scale":.2,"preview-src-list":[e(N)],src:e(N),"zoom-rate":1.2,fit:"contain",style:{"margin-top":"10px","margin-bottom":"10px"}},null,8,["preview-src-list","src"])])),y.value?(s(),r("div",ot,[l(k,{"hide-on-click-modal":!0,"initial-index":4,"max-scale":7,"min-scale":.2,"preview-src-list":[e(D)],src:e(D),"zoom-rate":1.2,fit:"contain",style:{"margin-top":"10px","margin-bottom":"10px"}},null,8,["preview-src-list","src"])])):g("",!0),i("div",it,n(e(t)("help.nine.text5")),1),y.value?g("",!0):(s(),r("div",st,[l(k,{"hide-on-click-modal":!0,"initial-index":4,"max-scale":7,"min-scale":.2,"preview-src-list":[e(C)],src:e(C),"zoom-rate":1.2,fit:"contain",style:{"margin-top":"10px","margin-bottom":"10px"}},null,8,["preview-src-list","src"])])),y.value?(s(),r("div",at,[l(k,{"hide-on-click-modal":!0,"initial-index":4,"max-scale":7,"min-scale":.2,"preview-src-list":[e(M)],src:e(M),"zoom-rate":1.2,fit:"contain",style:{"margin-top":"10px","margin-bottom":"10px"}},null,8,["preview-src-list","src"])])):g("",!0)]),_:1})]),_:1},8,["modelValue"])])]),_:1})]),_:1})]),_:1})])}}});export{ct as default};
