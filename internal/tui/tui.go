package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wdwb/tree-generator/internal/templates"
)

type simpleState int

const (
	stateName simpleState = iota
	stateDesc
	stateItem
	stateDone
)

type simpleModel struct {
	state    simpleState
	name     string
	desc     string
	items    []string
	input    textinput.Model
	viewport viewport.Model
}

func initialSimpleModel() simpleModel {
	ti := textinput.New()
	ti.Placeholder = "템플릿 이름을 입력하세요"
	ti.Focus()

	vp := viewport.New(80, 10)
	vp.SetContent("(비어 있음)")

	return simpleModel{
		state:    stateName,
		input:    ti,
		viewport: vp,
	}
}

func (m simpleModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case tea.KeyEnter:
			switch m.state {
			case stateName:
				if m.input.Value() != "" {
					m.name = m.input.Value()
					m.input.SetValue("")
					m.input.Placeholder = "템플릿 설명을 입력하세요"
					m.state = stateDesc
				}
			case stateDesc:
				m.desc = m.input.Value()
				m.input.SetValue("")
				m.input.Placeholder = "폴더/파일 이름을 입력하세요 (폴더는 a/ 처럼, 파일은 a.txt 처럼, 빈 입력시 종료)"
				m.state = stateItem
			case stateItem:
				val := m.input.Value()
				if val == "" {
					m.state = stateDone
					return m, tea.Quit
				}
				m.items = append(m.items, val)
				m.input.SetValue("")
				treeNodes := buildTree(m.items)
				treeStr := renderTreePreview(treeNodes, "")
				m.viewport.SetContent(treeStr)
			}
		}

	case tea.WindowSizeMsg:
		bottomAreaHeight := 8
		height := msg.Height - bottomAreaHeight
		if height < 3 {
			height = 3
		}
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = height
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m simpleModel) View() string {
	var b strings.Builder

	b.WriteString("\n[현재 입력 트리] (↑/↓/PgUp/PgDn 스크롤)\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	b.WriteString("[현재 입력 목록]\n")
	for _, item := range m.items {
		b.WriteString("  • " + item + "\n")
	}
	b.WriteString("\n")

	switch m.state {
	case stateName:
		b.WriteString("==== 템플릿 이름 입력 ====\n")
		b.WriteString(m.input.View())
	case stateDesc:
		b.WriteString("==== 템플릿 설명 입력 ====\n")
		b.WriteString(m.input.View())
	case stateItem:
		b.WriteString("==== 폴더/파일 이름을 입력하세요 (폴더는 a/ 처럼, 파일은 a.txt 처럼, 빈 입력시 종료) ====\n")
		b.WriteString(m.input.View())
	case stateDone:
		b.WriteString("\n저장 완료!")
	default:
		b.WriteString("오류 발생")
	}

	return b.String()
}

func buildTree(paths []string) []templates.TemplateNode {
	type node struct {
		Name     string
		Type     string
		Children map[string]*node
	}

	root := &node{Name: "", Type: "dir", Children: map[string]*node{}}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		isDirInput := strings.HasSuffix(path, "/")
		cleanPath := strings.TrimSuffix(path, "/")
		parts := strings.Split(filepath.ToSlash(cleanPath), "/")

		cur := root
		for i, part := range parts {
			if part == "" {
				continue
			}
			isLast := i == len(parts)-1
			nodeType := "file"
			if (isLast && isDirInput) || !isLast {
				nodeType = "dir"
			}

			if cur.Children == nil {
				cur.Children = map[string]*node{}
			}

			key := part + ":" + nodeType
			child, ok := cur.Children[key]
			if !ok {
				child = &node{Name: part, Type: nodeType, Children: map[string]*node{}}
				cur.Children[key] = child
			}

			if nodeType == "dir" {
				cur = child
			} else if isLast {
				break
			}
		}
	}

	var convert func(n *node) []templates.TemplateNode
	convert = func(n *node) []templates.TemplateNode {
		var result []templates.TemplateNode
		keys := make([]string, 0, len(n.Children))
		for k := range n.Children {
			keys = append(keys, k)
		}
		sort.SliceStable(keys, func(i, j int) bool {
			iParts := strings.Split(keys[i], ":")
			jParts := strings.Split(keys[j], ":")
			if iParts[1] != jParts[1] {
				return iParts[1] == "dir"
			}
			return iParts[0] < jParts[0]
		})

		for _, k := range keys {
			child := n.Children[k]
			tn := templates.TemplateNode{
				Name: child.Name,
				Type: child.Type,
			}
			if child.Type == "dir" && len(child.Children) > 0 {
				tn.Children = convert(child)
			}
			result = append(result, tn)
		}
		return result
	}
	return convert(root)
}

func renderTreePreview(nodes []templates.TemplateNode, prefix string) string {
	if len(nodes) == 0 {
		return "(비어 있음)"
	}
	var sb strings.Builder
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		connector := "├── "
		newPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		}
		sb.WriteString(prefix + connector + node.Name)
		if node.Type == "dir" {
		}
		sb.WriteString("\n")
		if node.Type == "dir" && len(node.Children) > 0 {
			sb.WriteString(renderTreePreview(node.Children, newPrefix))
		}
	}
	return sb.String()
}

func StartTUI() error {
	m := initialSimpleModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI 실행 중 오류 발생: %v", err)
	}
	if sm, ok := finalModel.(simpleModel); ok && sm.state == stateDone {
		tmpl := &templates.Template{
			Name:        sm.name,
			Description: sm.desc,
			Variables:   nil,
			Structure:   buildTree(sm.items),
		}
		return templates.SaveTemplate(tmpl)
	}
	return nil
}
