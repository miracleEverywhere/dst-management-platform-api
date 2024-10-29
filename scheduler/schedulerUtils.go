package scheduler

import (
	"bufio"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func getPlayersList() ([]string, error) {
	// 先执行命令
	_ = utils.BashCMD(utils.PlayersListCMD)
	// 获取日志文件中的list
	file, err := os.Open(utils.MasterLogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	var linesAfterKeyword []string
	var lines []string
	keyword := "playerlist 99999999 [0]"
	var foundKeyword bool

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		// 将行添加到结果切片
		linesAfterKeyword = append(linesAfterKeyword, line)

		// 检查是否包含关键字
		if strings.Contains(line, keyword) {
			foundKeyword = true
			break
		}
	}

	if !foundKeyword {
		return nil, fmt.Errorf("keyword not found in the file")
	}

	// 正则表达式匹配模式
	pattern := `playerlist 99999999 \[[0-9]+\] (KU_[A-Za-z0-9]+) (.+)`
	re := regexp.MustCompile(pattern)

	var players []string

	// 查找匹配的行并提取所需字段
	for _, line := range linesAfterKeyword {
		if matches := re.FindStringSubmatch(line); matches != nil {
			// 检查是否包含 [Host]
			if !regexp.MustCompile(`\[Host\]`).MatchString(line) {
				uid := strings.ReplaceAll(matches[1], "\t", "")
				uid = strings.ReplaceAll(uid, " ", "")
				nickName := strings.ReplaceAll(matches[2], "\t", "")
				nickName = strings.ReplaceAll(nickName, " ", "")

				player := uid + "," + nickName
				players = append(players, player)
			}
		}
	}

	players = utils.UniqueSliceKeepOrderString(players)

	return players, nil

}
