// Command codedoc 生成并校验业务错误码文档。
//
//	go run ./cmd/codedoc            # 生成 docs/error-codes.md
//	go run ./cmd/codedoc -verify    # 只校验，不写文件（CI 用）
//
// 校验项：码唯一、码落在所属模块号段内、词条能解析、每个码都有使用点。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dst-management-platform-api/constant"
	"dst-management-platform-api/i18n"
)

// i18n 词条表：错误码按模块归属，文案从对应模块表里查（基础文案已合并进每张表）。
var tables = map[string]*i18n.BaseI18n{
	"framework": &i18n.I18n,
	"user":      &i18n.User.BaseI18n,
	"room":      &i18n.Room.BaseI18n,
	"mod":       &i18n.Mod.BaseI18n,
	"dashboard": &i18n.Dashboard.BaseI18n,
	"platform":  &i18n.Platform.BaseI18n,
	"logs":      &i18n.Logs.BaseI18n,
	"tools":     &i18n.Tools.BaseI18n,
	"player":    &i18n.Player.BaseI18n,
	"cloud":     &i18n.Cloud.BaseI18n,
}

var moduleNames = map[string]string{
	"framework": "框架层", "user": "用户", "room": "房间", "mod": "模组", "dashboard": "控制台",
	"platform": "平台", "logs": "日志", "tools": "工具", "player": "玩家", "cloud": "云服务",
}

var levelNames = map[constant.Level]string{
	constant.LevelWarn: "warn", constant.LevelAuth: "auth", constant.LevelFatal: "fatal",
}

// 扫描范围：这些目录里的 constant.CodeXxx 引用会被统计为「出现位置」
var scanDirs = []string{"service", "app", "middleware"}

func main() {
	verifyOnly := flag.Bool("verify", false, "只校验不写文件")
	out := flag.String("out", filepath.Join("docs", "error-codes.md"), "输出路径")
	flag.Parse()

	usages, err := collectUsages()
	if err != nil {
		fail(err)
	}

	problems := verify(usages)
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "✗", p)
		}
		os.Exit(1)
	}

	if *verifyOnly {
		// 文档是否与码表一致（防止改了码表忘记重新生成）
		want := render(usages)
		if got, err := os.ReadFile(*out); err != nil {
			fmt.Fprintf(os.Stderr, "✗ 读不到 %s：%v；先运行 go run ./cmd/codedoc\n", *out, err)
			os.Exit(1)
		} else if string(got) != want {
			fmt.Fprintf(os.Stderr, "✗ %s 与码表不一致，运行 go run ./cmd/codedoc 重新生成\n", *out)
			os.Exit(1)
		}
		fmt.Printf("错误码校验通过：%d 个码，文案/使用点/号段正确，%s 已同步\n", len(constant.Registry), *out)
		return
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(*out, []byte(render(usages)), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("已生成 %s（%d 个码）\n", *out, len(constant.Registry))
}

// collectUsages 扫描源码，返回 常量名 -> 使用位置列表（文件:函数）。
func collectUsages() (map[string][]string, error) {
	usages := make(map[string][]string)
	fset := token.NewFileSet()
	for _, dir := range scanDirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			// 记录每个函数体内的 constant.Xxx 引用
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkg, ok := sel.X.(*ast.Ident)
					if !ok || pkg.Name != "constant" || !strings.HasPrefix(sel.Sel.Name, "Code") {
						return true
					}
					usages[sel.Sel.Name] = append(usages[sel.Sel.Name], path+":"+fn.Name.Name)
					return true
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for k, v := range usages {
		uniq := make(map[string]struct{}, len(v))
		var out []string
		for _, s := range v {
			if _, ok := uniq[s]; ok {
				continue
			}
			uniq[s] = struct{}{}
			out = append(out, s)
		}
		sort.Strings(out)
		usages[k] = out
	}
	return usages, nil
}

// verify 检查码表与文案、使用点的一致性。
func verify(usages map[string][]string) []string {
	var problems []string

	seen := make(map[int]string, len(constant.Registry))
	for _, e := range constant.Registry {
		if prev, ok := seen[e.Code]; ok {
			problems = append(problems, fmt.Sprintf("错误码 %d 重复（%s 与 %s）", e.Code, prev, e.TextKey))
		}
		seen[e.Code] = e.TextKey

		if lo, hi := codeRange(e.Module); lo != 0 && (e.Code < lo || e.Code > hi) {
			problems = append(problems, fmt.Sprintf("%d（%s）不在 %s 的号段 %d-%d 内", e.Code, e.TextKey, e.Module, lo, hi))
		}

		table, ok := tables[e.Module]
		if !ok {
			problems = append(problems, fmt.Sprintf("%d 的模块 %q 没有对应的 i18n 词条表", e.Code, e.Module))
			continue
		}
		if _, ok := table.ZH[e.TextKey]; !ok {
			problems = append(problems, fmt.Sprintf("%d 的词条 %q 在 %s 的中文表里不存在", e.Code, e.TextKey, e.Module))
		}
		if _, ok := table.EN[e.TextKey]; !ok {
			problems = append(problems, fmt.Sprintf("%d 的词条 %q 在 %s 的英文表里不存在", e.Code, e.TextKey, e.Module))
		}
		if strings.TrimSpace(e.Hint) == "" {
			problems = append(problems, fmt.Sprintf("%d 缺少 Hint（排查建议）", e.Code))
		}
	}

	// 反向：源码里引用的常量必须在注册表里有条目，且注册表条目必须被用到
	byName := make(map[string]constant.Entry, len(constant.Registry))
	for _, e := range constant.Registry {
		byName[e.Const] = e
	}
	for name, locs := range usages {
		if _, ok := byName[name]; !ok {
			problems = append(problems, fmt.Sprintf("源码引用了 %s，但注册表里没有对应条目（%s）", name, strings.Join(locs, ", ")))
		}
	}
	for name, e := range byName {
		if len(usages[name]) == 0 {
			problems = append(problems, fmt.Sprintf("%d（%s）在源码里没有任何使用点，属于死码", e.Code, e.TextKey))
		}
	}

	return problems
}

// codeRange 返回模块对应的号段，框架层单独一小段。
func codeRange(module string) (int, int) {
	order := []string{"user", "room", "mod", "dashboard", "platform", "logs", "tools", "player", "cloud"}
	for i, m := range order {
		if m == module {
			base := (i + 1) * 1000
			return base, base + 999
		}
	}
	if module == "framework" {
		return 0, 999
	}
	return 0, 0
}

func render(usages map[string][]string) string {
	var b strings.Builder
	b.WriteString("# 业务错误码\n\n")
	b.WriteString("> 本文件由 `go run ./cmd/codedoc` 自动生成，请勿手工编辑。\n")
	b.WriteString("> 新增错误码：在 `constant/code_registry.go` 登记条目，在 service 里引用 `constant.CodeXxx`，再重新生成。\n\n")
	b.WriteString("响应结构固定为 `{\"code\": ..., \"message\": ..., \"data\": ...}`；`code` 唯一标识一个错误原因，")
	b.WriteString("`message` 由 `i18n` 按 `code` 翻译（`X-I18n-Lang` 决定中英文）。成功固定 `200`，")
	b.WriteString("认证失效固定 `420`（前端据此清登录态）。\n\n")

	b.WriteString("## 号段\n\n| 号段 | 模块 |\n|---|---|\n")
	b.WriteString("| 0-999 | 框架层（token/权限/限流/参数） |\n")
	for i, m := range []string{"user", "room", "mod", "dashboard", "platform", "logs", "tools", "player", "cloud"} {
		base := (i + 1) * 1000
		fmt.Fprintf(&b, "| %d-%d | %s |\n", base, base+999, moduleNames[m])
	}

	b.WriteString("\n## 错误码一览\n\n")
	b.WriteString("| 码 | 模块 | 中文 | English | 级别 | 排查建议 | 出现位置 |\n")
	b.WriteString("|---:|---|---|---|---|---|---|\n")

	entries := append([]constant.Entry(nil), constant.Registry...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Code < entries[j].Code })
	for _, e := range entries {
		table := tables[e.Module]
		locs := usages[e.Const]
		loc := "—"
		if len(locs) > 0 {
			loc = strings.Join(locs, "<br>")
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s |\n",
			e.Code, moduleNames[e.Module], mdCell(table.ZH[e.TextKey]), mdCell(table.EN[e.TextKey]),
			levelNames[e.Level], mdCell(e.Hint), loc)
	}

	return b.String()
}

// mdCell 转义会破坏表格的字符
func mdCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "codedoc:", err)
	os.Exit(1)
}
