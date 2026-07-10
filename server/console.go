package server

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

type tableNamer interface {
	TableName() string
}

func runConsole(cmd, dbPath string) {
	fmt.Println("====== 饥荒管理平台 Console ======")
	fmt.Println()
	switch cmd {
	case "reset_password":
		resetPassword(dbPath)
	case "list_user":
		listUser(dbPath)
	case "db_stats":
		dbStats(dbPath)
	case "help":
		consoleInfo(cmd)
	default:
		consoleInfo(cmd)
	}
}

func initConsoleDB(dbPath string) {
	dbFile := filepath.Join(dbPath, "dmp.db")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		fmt.Printf("数据库文件不存在: %s\n", dbFile)
		fmt.Println("自定义数据库路径: ./dmp -dbpath <路径> -console <命令>")
		os.Exit(1)
	}
	logger.InitLogger("info")
	db.InitDB(dbPath)
}

func resetPassword(dbPath string) {
	initConsoleDB(dbPath)
	userDao := dao.NewUserDAO(db.DB)
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入用户名: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取用户名失败: %v\n", err)
		os.Exit(1)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		fmt.Println("用户名不能为空")
		os.Exit(1)
	}

	dbUser, err := userDao.GetUserByUsername(username)
	if err != nil {
		fmt.Printf("查询用户失败: %v\n", err)
		os.Exit(1)
	}
	if dbUser.Username == "" {
		fmt.Printf("用户 %s 不存在\n", username)
		os.Exit(1)
	}

	fmt.Print("请输入新密码: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("\n读取密码失败: %v\n", err)
		os.Exit(1)
	}
	password := strings.TrimSpace(string(passwordBytes))
	fmt.Println()
	if password == "" {
		fmt.Println("密码不能为空")
		os.Exit(1)
	}

	fmt.Print("请再次输入新密码: ")
	confirmBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("\n读取密码失败: %v\n", err)
		os.Exit(1)
	}
	confirm := strings.TrimSpace(string(confirmBytes))
	fmt.Println()
	if password != confirm {
		fmt.Println("两次输入的密码不一致")
		os.Exit(1)
	}

	hashedPassword, err := utils.GenerateBcryptPassword(password)
	if err != nil {
		logger.Logger.Errorf("创建bcrypt密码失败：%v", err)
		return
	}
	dbUser.Password = hashedPassword
	dbUser.PasswordVersion = models.PasswordVersionBcrypt

	err = userDao.UpdateUser(dbUser)
	if err != nil {
		fmt.Printf("更新密码失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("用户 %s 密码重置成功\n", username)
}

func listUser(dbPath string) {
	initConsoleDB(dbPath)
	userDao := dao.NewUserDAO(db.DB)

	users, err := userDao.ListUsers("", 1, 10000)
	if err != nil {
		fmt.Printf("查询用户列表失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-20s %-20s %-10s\n", "用户名", "昵称", "角色")
	fmt.Println(strings.Repeat("-", 55))
	for _, user := range users.Data {
		disabled := ""
		if user.Disabled {
			disabled = " (禁用)"
		}
		fmt.Printf("%-20s %-20s %-10s%s\n", user.Username, user.Nickname, user.Role, disabled)
	}
	fmt.Println(strings.Repeat("-", 55))
	fmt.Printf("共 %d 个用户\n", users.TotalCount)
}

func dbStats(dbPath string) {
	initConsoleDB(dbPath)

	dbFile := filepath.Join(dbPath, "dmp.db")
	fileInfo, err := os.Stat(dbFile)
	if err != nil {
		fmt.Printf("获取数据库文件信息失败: %v\n", err)
		os.Exit(1)
	}
	fileSize := fileInfo.Size()

	fmt.Printf("数据库文件: %s\n", dbFile)
	fmt.Printf("文件大小: %s\n\n", formatSize(fileSize))

	totalRows := int64(0)
	fmt.Printf("%-25s %10s\n", "表名", "行数")
	fmt.Println(strings.Repeat("-", 40))
	for _, m := range db.AllTables {
		var count int64
		db.DB.Model(m).Count(&count)
		tableName := m.(tableNamer).TableName()
		fmt.Printf("%-25s %10d\n", tableName, count)
		totalRows += count
	}
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("%-25s %10d\n", "合计", totalRows)
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func consoleInfo(cmd string) {
	if cmd != "help" {
		fmt.Printf("未知命令: %s\n\n", cmd)
		fmt.Println("可用命令:")
	}
	fmt.Println("  reset_password    重置用户密码")
	fmt.Println("  list_user         列出所有用户")
	fmt.Println("  db_stats          查看数据库统计")
	fmt.Println("  help              显示本信息")
	fmt.Println()
	os.Exit(1)
}
