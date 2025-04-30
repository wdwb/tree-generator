package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wdwb/tree-generator/internal/templates"
)

// Styles
var selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))                           // Green
var defaultInfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))                           // Gray
var selectedForDeleteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Strikethrough(true) // Red, Strikethrough
var cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))                                // Orange cursor

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

// --- Template Selection TUI ---

type selectModel struct {
	templates      []templates.Template
	cursor         int
	selected       string
	quitting       bool
	currentDefault string // 현재 기본 템플릿 이름 저장
}

func initialSelectModel(templates []templates.Template, currentDefault string) selectModel {
	return selectModel{
		templates:      templates,
		currentDefault: currentDefault, // 전달받은 기본값 저장
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if len(m.templates) > 0 && m.cursor >= 0 && m.cursor < len(m.templates) {
				m.selected = m.templates[m.cursor].Name
			}
			m.quitting = true
			return m, tea.Quit

		case "down", "j":
			if len(m.templates) > 0 {
				m.cursor = (m.cursor + 1) % len(m.templates)
			}

		case "up", "k":
			if len(m.templates) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.templates) - 1
				}
			}
		}
	}

	return m, nil
}

func (m selectModel) View() string {
	if m.quitting {
		if m.selected != "" {
			return fmt.Sprintf("\n선택된 템플릿: %s\n", m.selected)
		}
		return "\n템플릿 선택이 취소되었습니다.\n"
	}

	s := "어떤 템플릿을 기본으로 설정하시겠습니까?\n\n"

	for i, t := range m.templates {
		cursor := " "
		defaultIndicator := ""
		if t.Name == m.currentDefault {
			defaultIndicator = defaultInfoStyle.Render(" (default)")
		}
		line := fmt.Sprintf("[%s] (%s)%s", t.Name, t.Description, defaultIndicator)
		if m.cursor == i {
			cursor = ">"
			line = selectedItemStyle.Render(line)
		}
		s += fmt.Sprintf("%s %s\n", cursor, line)
	}

	s += "\n(↑/k: 위, ↓/j: 아래, Enter: 선택, q/Esc: 종료)\n"

	return s
}

// SelectTemplateTUI starts the TUI for selecting a template.
// It returns the selected template name or an empty string if canceled.
func SelectTemplateTUI(tmplList []templates.Template, currentDefault string) (string, error) {
	if len(tmplList) == 0 {
		return "", fmt.Errorf("선택할 템플릿이 없습니다")
	}

	m := initialSelectModel(tmplList, currentDefault) // 현재 기본값 전달
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI 실행 중 오류 발생: %w", err)
	}

	// Type assertion to get the final model state
	finalSelectModel, ok := finalModel.(selectModel)
	if !ok {
		return "", fmt.Errorf("최종 모델 타입 변환 실패")
	}

	return finalSelectModel.selected, nil // Return the selected name
}

// --- Template Deletion TUI ---

type deleteModel struct {
	templates []templates.Template
	cursor    int
	selected  map[int]bool // Map to track selected indices
	quitting  bool
}

func initialDeleteModel(templates []templates.Template) deleteModel {
	return deleteModel{
		templates: templates,
		selected:  make(map[int]bool),
	}
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.selected = make(map[int]bool) // Clear selection on quit
			m.quitting = true
			return m, tea.Quit

		case "enter":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if len(m.templates) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.templates) - 1
				}
			}

		case "down", "j":
			if len(m.templates) > 0 {
				m.cursor = (m.cursor + 1) % len(m.templates)
			}

		case " ": // Spacebar to toggle selection
			if len(m.templates) > 0 {
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor) // Deselect
				} else {
					m.selected[m.cursor] = true // Select
				}
			}
		}
	}
	return m, nil
}

func (m deleteModel) View() string {
	if m.quitting {
		if len(m.selected) > 0 {
			var names []string
			for idx := range m.selected {
				names = append(names, m.templates[idx].Name)
			}
			return fmt.Sprintf("\n삭제 예정: %v\n", names)
		}
		return "\n삭제가 취소되었습니다.\n"
	}

	s := "삭제할 템플릿을 선택하세요 (Space: 선택/해제, Enter: 확정)\n\n"

	for i, t := range m.templates {
		cursor := " "
		if m.cursor == i {
			cursor = cursorStyle.Render(">")
		}

		checked := "[ ]"
		lineStyle := lipgloss.NewStyle() // Default style
		if m.selected[i] {
			checked = selectedForDeleteStyle.Render("[x]")
			lineStyle = selectedForDeleteStyle
		}

		s += fmt.Sprintf("%s %s %s\n", cursor, checked, lineStyle.Render(fmt.Sprintf("%s (%s)", t.Name, t.Description)))
	}

	s += "\n(↑/k, ↓/j: 이동, Space: 선택/해제, Enter: 확정, q/Esc: 취소)\n"
	return s
}

// SelectTemplatesToDeleteTUI starts the TUI for selecting templates to delete.
// It returns a slice of template names selected for deletion.
func SelectTemplatesToDeleteTUI(tmplList []templates.Template) ([]string, error) {
	if len(tmplList) == 0 {
		return nil, fmt.Errorf("삭제할 템플릿이 없습니다")
	}

	m := initialDeleteModel(tmplList)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI 실행 중 오류 발생: %w", err)
	}

	finalDeleteModel, ok := finalModel.(deleteModel)
	if !ok {
		return nil, fmt.Errorf("최종 모델 타입 변환 실패")
	}

	var selectedNames []string
	if !finalDeleteModel.quitting || len(finalDeleteModel.selected) == 0 {
		// User quit without pressing Enter or selected nothing
		return nil, nil // Return empty list, no error
	}

	// Collect names based on selected indices
	for idx := range finalDeleteModel.selected {
		selectedNames = append(selectedNames, finalDeleteModel.templates[idx].Name)
	}
	sort.Strings(selectedNames) // Sort for predictable order
	return selectedNames, nil
}

// --- Existing TUI Code ---

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
			Variables:   extractVariables(sm.items),
			Structure:   buildTree(sm.items),
		}
		return templates.SaveTemplate(tmpl)
	}
	return nil
}

// extractVariables는 경로 목록에서 {변수명} 형식의 변수를 추출합니다.
func extractVariables(paths []string) []string {
	varRegex := regexp.MustCompile(`\{([^{}]+)\}`)
	varsMap := make(map[string]bool)

	for _, path := range paths {
		matches := varRegex.FindAllStringSubmatch(path, -1)
		for _, match := range matches {
			if len(match) > 1 {
				varsMap[match[1]] = true
			}
		}
	}

	// 맵의 키(변수명)를 슬라이스로 변환
	vars := make([]string, 0, len(varsMap))
	for k := range varsMap {
		vars = append(vars, k)
	}
	sort.Strings(vars) // 변수명을 알파벳 순으로 정렬
	return vars
}
