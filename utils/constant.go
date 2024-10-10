package utils

const MasterName = "Master"
const CavesName = "Caves"
const MasterScreenName = "DST_MASTER"
const CavesScreenName = "DST_CAVES"

const StartMasterCMD = "cd ~/dst/bin/ ; screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + MasterName + "  ;"

const StartCavesCMD = "cd ~/dst/bin/ ; screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + CavesName + "  ;"

const UpdateGameCMD = "cd ~/steamcmd ; ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"

const PlayersListCMD = "screen -S \"" + MasterScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s %s %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"

const MasterModPath = ".klei/DoNotStarveTogether/MyDediServer/" + MasterName + "/modoverrides.lua"

const CavesModPath = ".klei/DoNotStarveTogether/MyDediServer/" + CavesName + "/modoverrides.lua"

const MasterSettingPath = ".klei/DoNotStarveTogether/MyDediServer/" + MasterName + "/leveldataoverride.lua"

const CavesSettingPath = ".klei/DoNotStarveTogether/MyDediServer/" + CavesName + "/leveldataoverride.lua"

const ServerSettingPath = ".klei/DoNotStarveTogether/MyDediServer/cluster.ini"

const MasterLogPath = ".klei/DoNotStarveTogether/MyDediServer/" + MasterName + "/server_log.txt"

const CaveLogPath = ".klei/DoNotStarveTogether/MyDediServer/" + CavesName + "/server_log.txt"

const ChatLogPath = ".klei/DoNotStarveTogether/MyDediServer/" + MasterName + "/server_chat_log.txt"

const AdminListPath = ".klei/DoNotStarveTogether/MyDediServer/adminlist.txt"

const BlockListPath = ".klei/DoNotStarveTogether/MyDediServer/blocklist.txt"

const GameModSettingPath = "dst/mods/dedicated_server_mods_setup.lua"
