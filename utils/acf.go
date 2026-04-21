package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Node struct {
	Name     string
	Parent   *Node
	Children []*Node
	List     map[string]string
}

func NewNode(name string) *Node {
	return &Node{Name: name, List: make(map[string]string)}
}

func (n *Node) AddChild(name string) *Node {
	childNode := NewNode(name)
	childNode.Parent = n
	n.Children = append(n.Children, childNode)
	return childNode
}

func (n *Node) ReturnParent() *Node {
	if n.Parent != nil {
		return n.Parent
	}
	return nil
}

type Parser struct {
	Root   *Node
	Cursor *Node
}

func NewParser(path string) (*Parser, error) {
	f, err := os.Open(path)
	defer f.Close()

	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	p := &Parser{}

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	err = p.parse(lines)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func normalize(t string) string {
	return strings.Trim(strings.TrimSpace(t), `\t`)
}

func (p *Parser) parse(lines []string) error {
	// normalize
	for i, _ := range lines {
		lines[i] = normalize(lines[i])
	}

	// Parse Lines
	index := 0
	nodeLevel := 0
	r, _ := regexp.Compile(`\"([A-Za-z0-9\\\:\-\(\)\ \_\.]*)\"`)

loop:
	for {
		if index == len(lines) {
			if nodeLevel == 0 {
				break loop
			}
			return errors.New("Parser Error: Unmatched number of braces.")
		}

		line := lines[index]
		index++

		if line == "" {
			continue loop
		} else if r.MatchString(line) {
			matches := r.FindAllString(line, -1)
			if len(matches) == 2 {
				key := strings.Trim(matches[0], `"`)
				value := strings.Trim(matches[1], `"`)
				p.Cursor.List[key] = value
				continue loop
			} else if len(matches) == 1 && lines[index] == "{" {
				index++
				tag := strings.Trim(matches[0], `"`)
				if nodeLevel == 0 {
					p.Root = NewNode(tag)
					p.Cursor = p.Root
					nodeLevel++
				} else {
					p.Cursor = p.Cursor.AddChild(tag)
					nodeLevel++
				}
				continue loop
			} else {
				return errors.New("Parser Error: Contains the wrong string.")
			}
		} else if line == "}" {
			if p.Cursor.Parent != nil {
				p.Cursor = p.Cursor.ReturnParent()
			}
			nodeLevel--
			continue loop
		} else {
			return errors.New("Parser Error: Contains the wrong string.")
		}
	}
	return nil
}

func (p *Parser) GetWorkshopItemsInstalled(id string) (*Node, error) {
	if p == nil {
		return nil, fmt.Errorf("acf not parsed")
	}

	if p.Root.Name == "AppWorkshop" {
		for _, i := range p.Root.Children {
			if i.Name == "WorkshopItemsInstalled" {
				for _, j := range i.Children {
					if j.Name == id {
						return j, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("WorkshopItemsInstalled id not found")
}

func (p *Parser) AddWorkshopItemsInstalled(n *Node) error {
	if p == nil {
		return fmt.Errorf("acf not parsed")
	}

	var wsii *Node

	if p.Root.Name == "AppWorkshop" {
		for _, i := range p.Root.Children {
			if i.Name == "WorkshopItemsInstalled" {
				wsii = i
				break
			}
		}

		if wsii != nil {
			for _, i := range wsii.Children {
				if i.Name == n.Name {
					// 如果mod已存在，则删除
					err := p.RemoveWorkshopItemsInstalled(n.Name)
					if err != nil {
						return err
					}
				}
			}
			wsii.Children = append(wsii.Children, n)

			return nil
		}
	}

	return fmt.Errorf("WorkshopItemsInstalled not found")
}

func (p *Parser) RemoveWorkshopItemsInstalled(id string) error {
	if p == nil {
		return fmt.Errorf("acf not parsed")
	}

	if p.Root.Name == "AppWorkshop" {
		for indexI, i := range p.Root.Children {
			if i.Name == "WorkshopItemsInstalled" {
				for indexJ, j := range i.Children {
					if j.Name == id {
						p.Root.Children[indexI].Children = append(p.Root.Children[indexI].Children[:indexJ], p.Root.Children[indexI].Children[indexJ+1:]...)
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("WorkshopItemsInstalled id not found")
}

func (p *Parser) ListWorkshopItemsInstalled() []*Node {
	var data []*Node
	if p == nil || p.Root == nil || len(p.Root.Children) == 0 {
		return data
	}

	if p.Root.Name == "AppWorkshop" {
		for _, i := range p.Root.Children {
			if i.Name == "WorkshopItemsInstalled" {
				for _, j := range i.Children {
					data = append(data, j)
				}
			}
		}
	}

	return data
}

func (p *Parser) Format() []string {
	var (
		indent                 string
		acfContent             []string
		workshopItemsInstalled *Node
	)

	if p == nil {
		return []string{}
	}

	if p.Root.Name == "AppWorkshop" {
		indent = ""
		acfContent = append(acfContent, fmt.Sprintf("%s\"AppWorkshop\"", indent))
		acfContent = append(acfContent, fmt.Sprintf("%s{", indent))
		indent = "\t"

		for k, v := range p.Root.List {
			acfContent = append(acfContent, fmt.Sprintf("%s\"%s\"\t\t\"%s\"", indent, k, v))
		}

		for _, i := range p.Root.Children {
			if i.Name == "WorkshopItemsInstalled" {
				workshopItemsInstalled = i
				acfContent = append(acfContent, fmt.Sprintf("%s\"WorkshopItemsInstalled\"", indent))
				acfContent = append(acfContent, fmt.Sprintf("%s{", indent))

				for _, i := range workshopItemsInstalled.Children {
					indent = "\t\t"
					acfContent = append(acfContent, fmt.Sprintf("%s\"%s\"", indent, i.Name))
					acfContent = append(acfContent, fmt.Sprintf("%s{", indent))
					indent = "\t\t\t"
					for k, v := range i.List {
						acfContent = append(acfContent, fmt.Sprintf("%s\"%s\"\t\t\"%s\"", indent, k, v))
					}
					indent = "\t\t"
					acfContent = append(acfContent, fmt.Sprintf("%s}", indent))
				}

				indent = "\t"
				acfContent = append(acfContent, fmt.Sprintf("%s}", indent))

				break
			}
		}

		indent = ""
		acfContent = append(acfContent, fmt.Sprintf("%s}", indent))
	}

	return acfContent
}
