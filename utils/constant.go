package utils

const VERSION string = "1.1.7"

const MasterName = "Master"
const CavesName = "Caves"
const MasterScreenName = "DST_MASTER"
const CavesScreenName = "DST_CAVES"
const ServerPath = ".klei/DoNotStarveTogether/MyDediServer/"

const StartMasterCMD = "cd ~/dst/bin/ && screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + MasterName + "  ;"

const StartCavesCMD = "cd ~/dst/bin/ && screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + CavesName + "  ;"

const StartMaster64CMD = "cd ~/dst/bin64/ && screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster MyDediServer -shard " + MasterName + "  ;"

const StartCaves64CMD = "cd ~/dst/bin64/ && screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster MyDediServer -shard " + CavesName + "  ;"

const StopMasterCMD = "screen -S " + MasterScreenName + " -X quit"

const StopCavesCMD = "screen -S " + CavesScreenName + " -X quit"

const ClearScreenCMD = "screen -wipe"

const UpdateGameCMD = "cd ~/steamcmd ; ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"

const PlayersListMasterCMD = "screen -S \"" + MasterScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"

const PlayersListCavesCMD = "screen -S \"" + CavesScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"

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

const MasterBackupLogPath = ServerPath + MasterName + "/backup/server_log"

const CavesBackupLogPath = ServerPath + CavesName + "/backup/server_log"

const MasterChatLogPath = ServerPath + MasterName + "/server_chat_log.txt"

const CavesChatLogPath = ServerPath + CavesName + "/server_chat_log.txt"

const MasterBackupChatLogPath = ServerPath + MasterName + "/backup/server_chat_log"

const CavesBackupChatLogPath = ServerPath + CavesName + "/backup/server_chat_log"

const DMPLogPath = "./dmp.log"

const AdminListPath = ServerPath + "adminlist.txt"

const BlockListPath = ServerPath + "blocklist.txt"

const WhiteListPath = ServerPath + "whitelist.txt"

const GameModSettingPath = "dst/mods/dedicated_server_mods_setup.lua"

const MasterMetaPath = ServerPath + MasterName + "/save/session"

const CavesMetaPath = ServerPath + CavesName + "/save/session"

const DSTLocalVersionPath = "dst/version.txt"

const DSTServerVersionApi = "http://ver.tugos.cn/getLocalVersion"

const BackupPath = ".klei/DMP_BACKUP"

const KillDST = "pkill -f -9 dontstarve_dedicated_server_nullrenderer"

const InternetIPApi1 = "http://ip-api.com/json/?lang=zh-CN"

const InternetIPApi2 = "https://qifu-api.baidubce.com/ip/local/geo/v1/district"

const SteamApiKey = "1D15E021E1AB06D20F761C16525FFD40"

const SteamApiModDetail = "http://api.steampowered.com/IPublishedFileService/GetDetails/v1/"

const SteamApiModSearch = "http://api.steampowered.com/IPublishedFileService/QueryFiles/v1/"

const ProcessLogFile = "dmpProcess.log"

const ImportFileUploadPath = "/tmp/dst/"

const ImportFileUnzipPath = ImportFileUploadPath + "unzip/"

const MasterModUgcPath = "dst/ugc_mods/MyDediServer/Master/content/322330"

const CavesModUgcPath = "dst/ugc_mods/MyDediServer/Caves/content/322330"

const ModNoUgcPath = "dst/mods"

const ModDownloadPath = ".klei/DMP_MOD"

const NicknameUIDPath = ServerPath + "uid_map.json"

/* 以下是MacOS常量 Linux交叉编译：CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /root/dmp_darwin */

const MacStartMasterCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$HOME/steamcmd && screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + MasterName + "  ;"

const MacStartCavesCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$HOME/steamcmd && screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + CavesName + "  ;"

const MacGameModSettingPath = "dst/dontstarve_dedicated_server_nullrenderer.app/Contents/mods/dedicated_server_mods_setup.lua"

const MacModExportPath = "$HOME/Desktop/dmp_exported_mod"

const MacDSTVersionCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && strings dontstarve_dedicated_server_nullrenderer | grep -A 1 PRODUCTION | grep -E '\\d+'"
