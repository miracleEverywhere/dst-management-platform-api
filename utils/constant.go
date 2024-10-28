package utils

const MasterName = "Master"
const CavesName = "Caves"
const MasterScreenName = "DST_MASTER"
const CavesScreenName = "DST_CAVES"
const ServerPath = ".klei/DoNotStarveTogether/MyDediServer/"

const StartMasterCMD = "cd ~/dst/bin/ && screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + MasterName + "  ;"

const StartCavesCMD = "cd ~/dst/bin/ && screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + CavesName + "  ;"

const StopMasterCMD = "screen -S " + MasterScreenName + " -X quit"

const StopCavesCMD = "screen -S " + CavesScreenName + " -X quit"

const UpdateGameCMD = "cd ~/steamcmd ; ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"

const PlayersListCMD = "screen -S \"" + MasterScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s %s %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"

const MasterPath = ServerPath + MasterName

const CavesPath = ServerPath + CavesName

const MasterModPath = ServerPath + MasterName + "/modoverrides.lua"

const CavesModPath = ServerPath + CavesName + "/modoverrides.lua"

const MasterSettingPath = ServerPath + MasterName + "/leveldataoverride.lua"

const CavesSettingPath = ServerPath + CavesName + "/leveldataoverride.lua"

const ServerSettingPath = ServerPath + "cluster.ini"

const ServerTokenPath = ServerPath + "cluster_token.txt"

const MasterServerPath = ServerPath + MasterName + "/server.ini"

const CavesServerPath = ServerPath + CavesName + "/server.ini"

const MasterSavePath = ServerPath + MasterName + "/save"

const CavesSavePath = ServerPath + CavesName + "/save"

const MasterLogPath = ServerPath + MasterName + "/server_log.txt"

const CavesLogPath = ServerPath + CavesName + "/server_log.txt"

const ChatLogPath = ServerPath + MasterName + "/server_chat_log.txt"

const DMPLogPath = "./dmp.log"

const AdminListPath = ServerPath + "adminlist.txt"

const BlockListPath = ServerPath + "blocklist.txt"

const GameModSettingPath = "dst/mods/dedicated_server_mods_setup.lua"

const MetaPath = ServerPath + MasterName + "/save/session"

const DSTLocalVersionPath = "dst/version.txt"

const DSTServerVersionApi = "http://ver.tugos.cn/getLocalVersion"
