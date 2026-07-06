package dst

import "fmt"

// generateTmiCmd 根据命令类型生成对应的饥荒联机版 Lua 控制台命令。
//
// 参数说明:
//   - t:      命令类型，决定生成哪条 Lua 命令
//   - uid:    目标玩家的 Ku ID（UserToPlayer 所需的字符串标识）
//   - prefab: 通用参数，根据命令类型含义不同（物品名/季节名/皮肤名/坐标值等）
//   - num:    通用数量参数，根据命令类型含义不同（个数/百分比/等级/天数等）
func generateTmiCmd(t, uid, prefab string, num int) (string, string) {
	var (
		cmd  string
		cmd2 string
	)

	switch t {

	// ====================================================================
	// 一、物品生成类
	// ====================================================================

	case "generate":
		// 功能: 在指定玩家背包中生成物品，若物品不可携带则放在玩家脚下
		// uid:    目标玩家的 Ku ID
		// prefab: 要生成的物品预制物名称（如 "cutstone", "axe", "meat"）
		// num:    生成数量
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();for i=1,%d do local inst=SpawnPrefab('%s',nil,nil,uid);if inst.components.inventoryitem~=nil then player.components.inventory:GiveItem(inst) else inst.Transform:SetPosition(x,y,z) end;if inst.components.perishable then inst.components.perishable:SetPercent(1) end;if inst.components.finiteuses then inst.components.finiteuses:SetPercent(1) end;if inst.components.fueled then inst.components.fueled:SetPercent(1) end;if inst.components.temperature then inst.components.temperature:SetTemperature(25) end end end",
			uid, num, prefab,
		)

	case "generateComponents":
		// 功能: 获取指定物品的制作材料（查 AllRecipes 表，生成所有原材料放入背包）
		// uid:    目标玩家的 Ku ID
		// prefab: 要查询制作配方的物品名
		// num:    重复获取次数
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);local function tmi_give(item) if player~=nil and player.Transform then local x,y,z=player.Transform:GetWorldPosition();if item~=nil and item.components then if item.components.inventoryitem~=nil then if player.components and player.components.inventory then player.components.inventory:GiveItem(item) end else item.Transform:SetPosition(x,y,z) end end end end;local function tmi_mat(name) local recipe=AllRecipes[name];if recipe then for _,iv in pairs(recipe.ingredients) do for i=1,iv.amount do local item=SpawnPrefab(iv.type);tmi_give(item) end end end end;for i=1,%d or 1 do tmi_mat('%s') end",
			uid, num, prefab,
		)

	case "teleport":
		// 功能: 将玩家传送到地图上最近的指定 prefab 实体旁边
		// uid:    目标玩家的 Ku ID
		// prefab: 目标实体的预制物名称，可选项如下:
		//         地上世界:
		//           "multiplayer_portal"           - 绚丽之门（主）/ "multiplayer_portal_moonrock" - 月岩传送门（备选）
		//           "cave_entrance_open"           - 打开的洞穴入口（主）/ "cave_entrance" - 洞穴入口（备选）
		//           "pigking"                      - 猪王
		//           "moonbase"                     - 月基
		//           "oasislake"                    - 绿洲湖
		//           "critterlab"                   - 宠物实验室
		//           "chester_eyebone"              - 切斯特眼骨
		//           "stagehand"                    - 舞台之手
		//           "moon_fissure"                 - 月裂
		//           "beequeenhive"                 - 蜂后巢（主）/ "beequeenhivegrown" - 已生长的蜂后巢（备选）
		//           "klaus_sack"                   - 克劳斯袋
		//           "mooseegg"                     - 麋鹿蛋（主）/ "moose_nesting_ground" - 麋鹿巢（备选）
		//           "dragonfly"                    - 龙蝇（主）/ "dragonfly_spawner" - 龙蝇生成点（备选）
		//           "antlion"                      - 蚁狮（主）/ "antlion_spawner" - 蚁狮生成点（备选）
		//           "crabking"                     - 蟹王
		//           "hermitcrab"                   - 寄居蟹
		//           "walrus_camp"                  - 海象营地
		//           "statueglommer"                - 格罗门雕像
		//           "statuemaxwell"                - 麦斯威尔雕像
		//           "sculpture_rookhead"           - 战车头雕塑（主）/ "sculpture_knighthead" - 战马头（备选）/"sculpture_bishophead" - 主教头（备选）
		//           "sculpture_rookbody"           - 战车身雕塑（主）/ "sculpture_knightbody" - 战马身（备选）/"sculpture_bishopbody" - 主教身（备选）
		//           "moon_altar_rock_glass"        - 月坛玻璃石（主）/ "moon_altar_rock_idol" - 月坛神像石（备选）/"moon_altar_rock_seed" - 月坛种子石（备选）
		//           "lightninggoat"                - 电羊
		//           "beefalo"                      - 皮弗娄牛
		//           "deer"                         - 鹿
		//         洞穴世界:
		//           "cave_exit"                    - 洞穴出口
		//           "tentacle_pillar"              - 触手柱（主）/ "tentacle_pillar_hole" - 触手柱洞口（备选）
		//           "atrium_gate"                  - 中庭大门
		//           "ancient_altar"                - 远古祭坛（主）/ "ancient_altar_broken" - 损坏的远古祭坛（备选）
		//           "hutch_fishbowl"               - 哈奇鱼缸
		//           "minotaur"                     - 米诺陶（主）/ "minotaurchest" - 米诺陶宝箱（备选）
		//           "toadstool_cap"                - 蟾蜍王菌盖
		//           "rabbithouse"                  - 兔人房
		//           "monkeybarrel"                 - 猴桶
		//           "rocky"                        - 石虾
		// num:    未使用
		if prefab == "multiplayer_portal" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "multiplayer_portal_moonrock",
			)
		}
		if prefab == "cave_entrance_open" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "cave_entrance",
			)
		}
		if prefab == "beequeenhive" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "beequeenhivegrown",
			)
		}
		if prefab == "mooseegg" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "moose_nesting_ground",
			)
		}
		if prefab == "dragonfly" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "dragonfly_spawner",
			)
		}
		if prefab == "antlion" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "antlion_spawner",
			)
		}
		if prefab == "tentacle_pillar" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "tentacle_pillar_hole",
			)
		}
		if prefab == "ancient_altar" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "ancient_altar_broken",
			)
		}
		if prefab == "minotaur" {
			cmd2 = fmt.Sprintf(
				"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
				uid, "minotaurchest",
			)
		}

		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function tmi_goto(prefab) if player.Physics~=nil then player.Physics:Teleport(prefab.Transform:GetWorldPosition()) else player.Transform:SetPosition(prefab.Transform:GetWorldPosition()) end end;local target=c_findnext('%s');if target~=nil then tmi_goto(target) end end",
			uid, prefab,
		)

	case "teleportToPos":
		// 功能: 将玩家传送到指定坐标 (prefab 传入的坐标值作为 x 和 z, y 固定为 0)
		// uid:    目标玩家的 Ku ID
		// prefab: 目标 X/Z 坐标值（字符串形式，如 "350.5"）
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then if player.Physics~=nil then player.Physics:Teleport(%s,0,%s) else player.Transform:SetPosition(%s,0,%s) end end",
			uid, prefab, prefab, prefab, prefab,
		)

	case "deleteItem":
		// 功能: 删除玩家周围指定半径内所有指定类型的实体
		// uid:    目标玩家的 Ku ID
		// prefab: 要删除的实体预制物名称
		// num:    搜索半径（游戏内距离单位，默认建议 3-30）
		cmd = fmt.Sprintf(
			"local uid='%s';local a=UserToPlayer(uid);local function b(c) local d=c.components.inventoryitem;return d and d.owner and true or false end;local function e(f) if f and f~=TheWorld and not b(f) and f.Transform then if f:HasTag('player') then if f.userid==nil or f.userid=='' then return true end else return true end end;return false end;if a and a.Transform then if a.components.burnable then a.components.burnable:Extinguish(true) end;local g,h,i=a.Transform:GetWorldPosition();local j=TheSim:FindEntities(g,h,i,%d);for k,l in pairs(j) do if e(l) then if l.components then if l.components.burnable then l.components.burnable:Extinguish(true) end;if l.components.firefx then if l.components.firefx.extinguishsoundtest then l.components.firefx.extinguishsoundtest=function() return true end end;l.components.firefx:Extinguish() end end;if l.prefab=='%s' then l:Remove() end end end end",
			uid, num, prefab,
		)

	case "deleteAround":
		// 功能: 删除玩家周围 3 范围内所有实体（排除光源和玩家背包内物品）
		// uid:    目标玩家的 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);local function InInv(b) local inv=b.components.inventoryitem;return inv and inv.owner and true or false end;local function CanDelete(inst) if inst and inst~=TheWorld and not InInv(inst) and inst.Transform then if inst:HasTag('player') then if inst.userid==nil or inst.userid=='' then return true end else return true end end;return false end;if player and player.Transform then if player.components.burnable then player.components.burnable:Extinguish(true) end;local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,3);for _,obj in pairs(ents) do if CanDelete(obj) then if obj.components then if obj.components.burnable then obj.components.burnable:Extinguish(true) end;if obj.components.firefx then if obj.components.firefx.extinguishsoundtest then obj.components.firefx.extinguishsoundtest=function() return true end end;obj.components.firefx:Extinguish() end end;if (not(obj.prefab=='minerhatlight' or obj.prefab=='lanternlight' or obj.prefab=='yellowamuletlight' or obj.prefab=='slurperlight' or obj.prefab=='redlanternlight' or obj.prefab=='lighterfire' or obj.prefab=='torchfire' or obj.prefab=='torchfire_rag' or obj.prefab=='torchfire_spooky' or obj.prefab=='torchfire_shadow')) or (obj.entity:GetParent()==nil) then obj:Remove() end end end end",
			uid,
		)

	// ====================================================================
	// 二、角色状态类
	// ====================================================================

	case "hunger":
		// 功能: 设置玩家的饥饿值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    饥饿值百分比（0-100, 100=满饥饿, 0或负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.hunger then player.components.hunger:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "sanity":
		// 功能: 设置玩家的理智值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    理智值百分比（0-100, 100=满理智, 0或负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.sanity then player.components.sanity:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "health":
		// 功能: 设置玩家的生命值（受生命惩罚上限影响）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    生命值百分比（0-100, 100=满血, 0或负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.health then player.components.health:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "healthLock":
		// 功能: 切换生命值锁定状态（切换 minhealth 在 0 和 1 之间，锁定后血量不会降到 0 以下）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.health then local h=player.components.health;local hpper=h:GetPercent();local minhp=h.minhealth;if minhp==0 then h:SetMinHealth(1) else h:SetMinHealth(0) end;h:SetPercent(hpper) end",
			uid,
		)

	case "moisture":
		// 功能: 设置玩家的潮湿度
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    潮湿度百分比（0-100, 0=干燥, 负数=默认0%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.moisture then player.components.moisture:SetPercent(%s) end",
			uid, formatFloat(num, 0),
		)

	case "playerTemperature":
		// 功能: 设置玩家的体温
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    体温值（摄氏度, 正常约 25, 过热 70+, 过冷 0-）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.temperature then player.components.temperature:SetTemperature(%d) end",
			uid, num,
		)

	case "inspiration":
		// 功能: 设置薇格弗德（Wigfrid）的歌唱灵感值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    灵感值百分比（0-100, 负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.singinginspiration then player.components.singinginspiration:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	// ====================================================================
	// 三、特殊模式类
	// ====================================================================

	case "godMode":
		// 功能: 切换上帝模式（SetInvincible 切换，幽灵状态时自动复活）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then if player:HasTag('playerghost') then player:PushEvent('respawnfromghost');player.rezsource='饥荒管理平台(DMP)' else if player.components.health~=nil then local godmode=player.components.health.invincible;player.components.health:SetInvincible(not godmode) end end end",
			uid,
		)

	case "creativeMode":
		// 功能: 切换创造模式（解锁所有制作配方，不需要材料即可制作）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.builder then player.components.builder:GiveAllRecipes() end",
			uid,
		)

	case "oneHitKill":
		// 功能: 切换一击必杀模式（替换 CalcDamage 函数使伤害为极大值）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.combat and player.components.combat.CalcDamage then local c=player.components.combat;if c.OldCalcDamage then c.CalcDamage=c.OldCalcDamage;c.OldCalcDamage=nil else c.OldCalcDamage=c.CalcDamage;c.CalcDamage=function(...) return 9999999999*9 end end end",
			uid,
		)

	case "invisible":
		// 功能: 切换隐身模式（添加/移除 debugnoattack 标签，生物不会主动攻击）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then if player:HasTag('debugnoattack') then player:RemoveTag('debugnoattack') else player:AddTag('debugnoattack') end end",
			uid,
		)

	// ====================================================================
	// 四、背包管理类
	// ====================================================================

	case "clearInventory":
		// 功能: 清空玩家物品栏中所有物品（直接 Remove）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    0=仅清物品栏, 1=同时清空背包中的物品
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local inventory=player.components.inventory;local backpack=inventory:GetOverflowContainer();local invSlots=inventory:GetNumSlots();local bpSlots=backpack and backpack:GetNumSlots() or 0;local removeAll=%d;for i=1,invSlots do local item=inventory:GetItemInSlot(i);inventory:RemoveItem(item,true);if item~=nil then item:Remove() end end;if removeAll==1 then for i=1,bpSlots do local item=backpack:GetItemInSlot(i);inventory:RemoveItem(item,true);if item~=nil then item:Remove() end end end end",
			uid, num,
		)

	case "clearBackpack":
		// 功能: 清空玩家背包中所有物品
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    0=仅清背包, 1=同时清空物品栏中的物品
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local inventory=player.components.inventory;local backpack=inventory:GetOverflowContainer();local invSlots=inventory:GetNumSlots();local bpSlots=backpack and backpack:GetNumSlots() or 0;local removeAll=%d;for i=1,bpSlots do local item=backpack:GetItemInSlot(i);inventory:RemoveItem(item,true);if item~=nil then item:Remove() end end;if removeAll==1 then for i=1,invSlots do local item=inventory:GetItemInSlot(i);inventory:RemoveItem(item,true);if item~=nil then item:Remove() end end end end",
			uid, num,
		)

	// ====================================================================
	// 五、角色专属类
	// ====================================================================

	case "wereBeaver":
		// 功能: 让伍迪（Woodie）变身为海狸形态并设置变身值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    变身值百分比（0-100, 负数=默认100%满变身值）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.wereness then player.components.wereness:SetWereMode('beaver');player.components.wereness:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "wereGoose":
		// 功能: 让伍迪（Woodie）变身为大鹅形态并设置变身值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    变身值百分比（0-100, 负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.wereness then player.components.wereness:SetWereMode('goose');player.components.wereness:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "wereMoose":
		// 功能: 让伍迪（Woodie）变身为麋鹿形态并设置变身值
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    变身值百分比（0-100, 负数=默认100%）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.wereness then player.components.wereness:SetWereMode('moose');player.components.wereness:SetPercent(%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "abigailLevel":
		// 功能: 设置温蒂（Wendy）的阿比盖尔（Abigail）羁绊等级
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    等级（1-3），数值越大阿比盖尔越强
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.ghostlybond then player.components.ghostlybond:SetBondLevel(%d) end",
			uid, num,
		)

	case "bloomLevel":
		// 功能: 设置沃姆伍德（Wormwood）的开花等级
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    开花等级（0-3），0=不开发, 3=满开
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player and not player:HasTag('playerghost') and player.components.bloomness then player.components.bloomness:SetLevel(%d) end",
			uid, num,
		)

	// ====================================================================
	// 六、季节/时间/天气类
	// ====================================================================

	case "season":
		// 功能: 设置当前世界季节
		// uid:    未使用
		// prefab: 季节名（"spring", "summer", "autumn", "winter"）
		// num:    未使用
		cmd = fmt.Sprintf("TheWorld:PushEvent('ms_setseason','%s')", prefab)

	case "nextPhase":
		// 功能: 跳到当前时间的下一阶段（白天→黄昏→夜晚→白天...）
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "TheWorld:PushEvent('ms_nextphase')"

	case "skipDays":
		// 功能: 跳过指定天数
		// uid:    未使用
		// prefab: 未使用
		// num:    要跳过的天数
		cmd = fmt.Sprintf("c_skip(%d)", num)

	case "timeScale":
		// 功能: 设置游戏时间倍速
		// uid:    未使用
		// prefab: 未使用
		// num:    倍速百分比（100=1.0x 正常, 200=2.0x, 50=0.5x 慢速）
		cmd = fmt.Sprintf("TheSim:SetTimeScale(%s)", formatFloat(num, 1.0))

	case "speed":
		// 功能: 设置玩家的移动速度倍率
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    速度倍率百分比（100=1.0x 正常, 200=2.0x, 60=0.6x, 负数=默认正常1.0）
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.locomotor then player.components.locomotor:SetExternalSpeedMultiplier(player,'c_speedmult',%s) end",
			uid, formatFloat(num, 1.0),
		)

	case "lightning":
		// 功能: 在玩家位置召唤一道闪电
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then TheWorld:PushEvent('ms_sendlightningstrike',Vector3(player.Transform:GetWorldPosition())) end",
			uid,
		)

	case "rainOn":
		// 功能: 强制开始下雨
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "TheWorld:PushEvent('ms_forceprecipitation',true)"

	case "rainOff":
		// 功能: 强制停止下雨
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "TheWorld:PushEvent('ms_forceprecipitation',false)"

	// ====================================================================
	// 七、角色操作类
	// ====================================================================

	case "kill":
		// 功能: 杀死目标玩家（触发 death 事件）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then player:PushEvent('death');player.deathpkname='饥荒管理平台(DMP)' end",
			uid,
		)

	case "resurrect":
		// 功能: 复活目标玩家（从幽灵状态复活）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then player:PushEvent('respawnfromghost');player.rezsource='饥荒管理平台(DMP)' end",
			uid,
		)

	case "despawn":
		// 功能: 将目标玩家从世界中移除（c_despawn）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);c_despawn(player)",
			uid,
		)

	case "gatherPlayers":
		// 功能: 将服务器上所有玩家传送到目标玩家身边
		// uid:    目标玩家 Ku ID（传送目的地）
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();for k,v in pairs(AllPlayers) do v.Transform:SetPosition(x,y,z) end end",
			uid,
		)

	case "unlockTech":
		// 功能: 解锁目标玩家的所有科技配方（科学/魔法/远古/暗影/制图各 10 级）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.builder then player.components.builder:UnlockRecipesForTech({SCIENCE=10,MAGIC=10,ANCIENT=10,SHADOW=10,CARTOGRAPHY=10}) end",
			uid,
		)

	case "penaltyAdd":
		// 功能: 增加目标玩家的生命上限惩罚 0.25（降低最大生命值）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.health then player.components.health:SetPenalty(player.components.health.penalty+0.25);player.components.health:ForceUpdateHUD(true) end",
			uid,
		)

	case "penaltyReduce":
		// 功能: 减少目标玩家的生命上限惩罚 0.25（恢复最大生命值）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and player.components.health then player.components.health:SetPenalty(player.components.health.penalty-0.25);player.components.health:ForceUpdateHUD(true) end",
			uid,
		)

	// ====================================================================
	// 八、实体操作类
	// ====================================================================

	case "extinguish":
		// 功能: 熄灭玩家周围 30 范围内所有火焰（保留营火/火坑等固定火源）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local a=UserToPlayer(uid);if a~=nil then local function b(c) local d=c.components.inventoryitem;return d and d.owner and true or false end;local function e(f) if f and f~=TheWorld and not b(f) and f.Transform then if f:HasTag('player') then if f.userid==nil or f.userid=='' then return true end else if f.prefab and f.prefab~='campfire' and f.prefab~='firepit' and f.prefab~='coldfire' and f.prefab~='coldfirepit' and f.prefab~='nightlight' then return true end end end;return false end;if a and a.Transform then if a.components.burnable then a.components.burnable:Extinguish(true) end;local g,h,i=a.Transform:GetWorldPosition();local j=TheSim:FindEntities(g,h,i,30);for k,l in pairs(j) do if e(l) then if l.components then if l.components.burnable then l.components.burnable:Extinguish(true) end;if l.components.firefx then if l.components.firefx.extinguishsoundtest then l.components.firefx.extinguishsoundtest=function() return true end end;l.components.firefx:Extinguish() end end end end end",
			uid,
		)

	case "fertilize":
		// 功能: 对玩家周围 30 范围内所有可施肥的植物使用便便施肥
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30);local poop=nil;for k,obj in pairs(ents) do if not obj:HasTag('player') and obj~=TheWorld and obj.AnimState and obj.Transform then if not(poop and poop.components and poop.components.fertilizer) then poop=c_spawn('poop') end;if obj and obj.components.crop and not obj.components.crop:IsReadyForHarvest() and not obj:HasTag('withered') then obj.components.crop:Fertilize(poop) elseif obj.components.grower and obj.components.grower:IsEmpty() then obj.components.grower:Fertilize(poop) elseif obj.components.pickable and obj.components.pickable:CanBeFertilized() then obj.components.pickable:Fertilize(poop) end end end;if poop~=nil then poop:Remove() end end",
			uid,
		)

	case "growth":
		// 功能: 催熟玩家周围 30 范围内随机一个植物（支持作物/树/蘑菇农场等）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local function trygrowth(inst) if inst:IsInLimbo() or (inst.components.witherable~=nil and inst.components.witherable:IsWithered()) then return end;if inst.components.pickable~=nil then if inst.components.pickable:CanBePicked() and inst.components.pickable.caninteractwith then return end;inst.components.pickable:FinishGrowing() end;if inst.components.crop~=nil then inst.components.crop:DoGrow(TUNING.TOTAL_DAY_TIME*3,true) end;if inst.components.growable~=nil and inst:HasTag('tree') and not inst:HasTag('stump') then inst.components.growable:DoGrowth() end;if inst.components.harvestable~=nil and inst.components.harvestable:CanBeHarvested() and inst:HasTag('mushroom_farm') then inst.components.harvestable:Grow() end end;local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30,nil,{'pickable','stump','withered','INLIMBO'});if #ents>0 then trygrowth(table.remove(ents,math.random(#ents)));if #ents>0 then local timevar=1-1/(#ents+1);for i,v in ipairs(ents) do v:DoTaskInTime(timevar*math.random(),trygrowth) end end end end",
			uid,
		)

	case "harvest":
		// 功能: 收割玩家周围 30 范围内所有可收获的作物/晾肉架/烹饪锅等
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and not player:HasTag('playerghost') then local function tryharvest(inst) local objc=inst.components;if objc.crop~=nil then objc.crop:Harvest(player) elseif objc.harvestable~=nil then objc.harvestable:Harvest(player) elseif objc.stewer~=nil then objc.stewer:Harvest(player) elseif objc.dryer~=nil then objc.dryer:Harvest(player) elseif objc.occupiable~=nil and objc.occupiable:IsOccupied() then local item=objc.occupiable:Harvest(player);if item~=nil then player.components.inventory:GiveItem(item) end elseif objc.pickable~=nil and objc.pickable:CanBePicked() then objc.pickable:Pick(player) end end;local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30);for k,obj in pairs(ents) do if not obj:HasTag('player') and not obj:HasTag('flower') and not obj:HasTag('trap') and not obj:HasTag('mine') and not obj:HasTag('cage') and obj~=TheWorld and obj.AnimState and obj.components and obj.prefab and not string.find(obj.prefab,'mandrake') then tryharvest(obj) end end end",
			uid,
		)

	case "pickup":
		// 功能: 自动捡取玩家周围 30 范围内所有可拾取物品放入背包
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and not player:HasTag('playerghost') then local inv=player.components.inventory;local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30,{'_inventoryitem'},{'INLIMBO','NOCLICK','catchable','fire'});local baits={['powcake']=true,['pigskin']=true,['winter_food4']=true};local function Wall(item) local xx,yy,zz=item.Transform:GetWorldPosition();local nents=TheSim:FindEntities(xx,yy,zz,3);local targets=0;for _,vv in ipairs(nents) do if vv:HasTag('wall') and vv.components.health then targets=targets+1 end end;return targets end;for _,v in ipairs(ents) do local c=v.components;if c.inventoryitem~=nil and c.inventoryitem.canbepickedup and c.inventoryitem.cangoincontainer and not c.inventoryitem:IsHeld() and not v:HasTag('flower') and not v:HasTag('trap') and not v:HasTag('mine') and not v:HasTag('cage') and not string.find(v.prefab,'mooneye') and inv and inv:CanAcceptCount(v,1)>0 then if c.trap~=nil and c.trap:IsSprung() then c.trap:Harvest(player) else if baits[v.prefab] then if Wall(v)<7 then inv:GiveItem(v) end else if c.bait then if not c.bait.trap then inv:GiveItem(v) end else inv:GiveItem(v) end end end end end end",
			uid,
		)

	case "freeze":
		// 功能: 冰冻玩家周围 15 范围内所有可冻结的生物
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,15);for k,obj in pairs(ents) do if not obj:HasTag('player') and obj~=TheWorld and obj.AnimState and obj.Transform and obj.components and obj.components.freezable~=nil then obj.components.freezable:AddColdness(1,60);obj.components.freezable:SpawnShatterFX() end end end",
			uid,
		)

	// ====================================================================
	// 九、皮弗娄牛类
	// ====================================================================

	case "beefalo":
		// 功能: 在玩家位置生成一头已驯化的皮弗娄牛
		// uid:    目标玩家 Ku ID
		// prefab: 皮弗娄牛倾向（TENDENCY），可选项:
		//           "TENDENCY.ORNERY"  - 战牛
		//           "TENDENCY.RIDER"   - 骑牛
		//           "TENDENCY.PUDGY"   - 肥牛
		//           "TENDENCY.DEFAULT" - 默认
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local beef=c_spawn('beefalo');beef.components.domesticatable:DeltaDomestication(1);beef.components.domesticatable:DeltaObedience(1);beef.components.domesticatable:DeltaTendency(%s,1);beef:SetTendency();beef.components.domesticatable:BecomeDomesticated();beef.components.hunger:SetPercent(0.5);beef.Transform:SetPosition(x,y,z) end",
			uid, prefab,
		)

	// ====================================================================
	// 十、跟随者类
	// ====================================================================

	case "followerAdd":
		// 功能: 将玩家周围 5 范围内所有生物强制收为跟随者（6000 秒忠诚时间）
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and not player:HasTag('playerghost') then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,5);for k,obj in pairs(ents) do if not obj:HasTag('player') and obj~=TheWorld and obj.AnimState and obj.Transform and obj.components and obj.components.follower~=nil then if obj.components.combat and obj.components.combat:TargetIs(player) then obj.components.combat:SetTarget(nil) end;if player.components.leader~=nil then player:PushEvent('makefriend');player.components.leader:AddFollower(obj);obj.components.follower:AddLoyaltyTime(6000);obj.components.follower.maxfollowtime=6000 end end end end",
			uid,
		)

	case "followerExpel":
		// 功能: 驱逐玩家周围 8 范围内所有跟随者
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,8);for k,obj in pairs(ents) do if obj.components and obj.components.follower~=nil and player.components.leader~=nil and player.components.leader:IsFollower(obj) then player.components.leader:RemoveFollower(obj);obj.components.follower.targettime=0 end end end",
			uid,
		)

	case "followerHealth":
		// 功能: 将玩家周围 30 范围内所有跟随者的生命值回满
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30);for k,obj in pairs(ents) do if obj.components and obj.components.follower~=nil and player.components.leader~=nil and player.components.leader:IsFollower(obj) and obj.components.health then obj.components.health:SetPercent(1) end end end",
			uid,
		)

	case "followerHunger":
		// 功能: 将玩家周围 30 范围内所有跟随者的饥饿值回满
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30);for k,obj in pairs(ents) do if obj.components and obj.components.follower~=nil and player.components.leader~=nil and player.components.leader:IsFollower(obj) and obj.components.hunger then obj.components.hunger:SetPercent(1) end end end",
			uid,
		)

	case "followerLoyal":
		// 功能: 将玩家周围 30 范围内所有跟随者的忠诚度回满
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local player=UserToPlayer(uid);if player~=nil and not player:HasTag('playerghost') then local x,y,z=player.Transform:GetWorldPosition();local ents=TheSim:FindEntities(x,y,z,30);for k,obj in pairs(ents) do if obj.components and obj.components.follower~=nil and player.components.leader~=nil and player.components.leader:IsFollower(obj) then obj.components.follower.targettime=obj.components.follower.maxfollowtime+GetTime();if obj.components.domesticatable then obj.components.domesticatable:DeltaObedience(1) end end end end",
			uid,
		)

	// ====================================================================
	// 十一、服务器管理类
	// ====================================================================

	case "regenerateWorld":
		// 功能: 重新生成世界地图（保留玩家数据，重新生成地形）
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "c_regenerateworld()"

	case "save":
		// 功能: 强制保存游戏
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "c_save()"

	// ====================================================================
	// 十二、活动事件类
	// ====================================================================

	case "specialEvent":
		// 功能: 切换世界特殊活动事件（应用后 5 秒自动回档使设置生效）
		// uid:    未使用
		// prefab: 活动事件名:
		//         "none"               - 无活动
		//         "default"            - 默认
		//         "hallowed_nights"    - 万圣夜
		//         "winters_feast"      - 冬季盛宴
		//         "year_of_the_gobbler" - 火鸡年
		//         "year_of_the_varg"   - 座狼年
		//         "year_of_the_pig"    - 猪年
		//         "year_of_the_carrat" - 胡萝卜鼠年
		//         "year_of_the_beefalo"- 皮弗娄牛年
		// num:    未使用
		cmd = fmt.Sprintf(
			"ApplySpecialEvent('%s');TheWorld.topology.overrides.specialevent='%s';c_save();TheWorld:DoTaskInTime(5,function() if TheWorld~=nil and TheWorld.ismastersim then TheNet:SendWorldRollbackRequestToServer(0) end end)",
			prefab, prefab,
		)

	// ====================================================================
	// 十三、地图类
	// ====================================================================

	case "revealMap":
		// 功能: 为指定玩家全开地图迷雾
		// uid:    目标玩家 Ku ID
		// prefab: 未使用
		// num:    未使用
		cmd = fmt.Sprintf(
			"local uid='%s';local p=UserToPlayer(uid);local m=p and p.userid and p.player_classified and p.player_classified.MapExplorer;if m then local w,h=TheWorld.Map:GetSize();for x=-w*2,w*2,10 do for z=-h*2,h*2,10 do if TheWorld.Map:IsValidTileAtPoint(x,0,z) then m:RevealArea(x,0,z) end end end end",
			uid,
		)

	case "clearMap":
		// 功能: 清除小地图上所有已探索区域（恢复迷雾）
		// uid:    未使用
		// prefab: 未使用
		// num:    未使用
		cmd = "TheWorld.minimap.MiniMap:ClearRevealedAreas()"
	}

	return cmd, cmd2
}

// formatFloat 将 int 类型的百分比值转为 Lua 可用的浮点数字符串。
// num <= 0 时返回默认值 defaultVal，否则返回 num/100 的浮点字符串（如 num=80 → "0.80"）。
func formatFloat(num int, defaultVal float64) string {
	if num <= 0 {
		return fmt.Sprintf("%.2f", defaultVal)
	}
	return fmt.Sprintf("%.2f", float64(num)/100.0)
}

func (g *Game) tmiConsoleCmd(t, uid, prefab string, num, worldID int) error {
	cmd, cmd2 := generateTmiCmd(t, uid, prefab, num)
	if cmd2 != "" {
		_ = g.consoleCmd(cmd2, worldID)
	}

	return g.consoleCmd(cmd, worldID)
}
