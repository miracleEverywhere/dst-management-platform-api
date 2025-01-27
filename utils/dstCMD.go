package utils

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

const KillDST = "pkill -f -9 dontstarve_dedicated_server_nullrenderer"

const ShardSession = "TheNet:GetUserSessionFile(ShardGameIndex:GetSession()"

const UserDataEncode = "TheNet:GetDefaultEncodeUserPath()"

/* 以下是MacOS常量 Linux交叉编译：CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /root/dmp_darwin */

const MacStartMasterCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$HOME/steamcmd && screen -d -m -S \"" + MasterScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + MasterName + "  ;"

const MacStartCavesCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$HOME/steamcmd && screen -d -m -S \"" + CavesScreenName + "\"  ./dontstarve_dedicated_server_nullrenderer -console -cluster MyDediServer -shard " + CavesName + "  ;"

const MacDSTVersionCMD = "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && strings dontstarve_dedicated_server_nullrenderer | grep -A 1 PRODUCTION | grep -E '\\d+'"
