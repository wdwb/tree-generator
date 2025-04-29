package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
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
	state simpleState
	name  string
	desc  string
	items []string
	input textinput.Model
	err   error
}

func initialSimpleModel() simpleModel {
	ti := textinput.New()
	ti.Placeholder = "템플릿 이름을 입력하세요"
	ti.Focus()
	return simpleModel{
		state: stateName,
		input: ti,
	}
}

func (m simpleModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
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
				m.input.Placeholder = "폴더/파일 이름을 입력하세요 (빈 입력시 종료)"
				m.state = stateItem
			case stateItem:
				val := m.input.Value()
				if val == "" {
					m.state = stateDone
					return m, tea.Quit
				}
				m.items = append(m.items, val)
				m.input.SetValue("")
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m simpleModel) View() string {
	switch m.state {
	case stateName:
		return "[템플릿 이름]" + "\n" + m.input.View()
	case stateDesc:
		return "[템플릿 설명]" + "\n" + m.input.View()
	case stateItem:
		return "[폴더/파일 이름 입력] (빈 입력시 종료)\n" + m.input.View() + "\n현재 입력: " + fmt.Sprintf("%v", m.items)
	case stateDone:
		return "저장 완료!"
	default:
		return "오류 발생"
	}
}

// StartTUI는 아주 단순한 템플릿 생성 TUI를 시작합니다
func StartTUI() error {
	m := initialSimpleModel()
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI 실행 중 오류 발생: %v", err)
	}
	// 저장
	if sm, ok := finalModel.(simpleModel); ok && sm.state == stateDone {
		tmpl := &templates.Template{
			Name:        sm.name,
			Description: sm.desc,
			Variables:   nil,
			Structure:   []templates.TemplateNode{},
		}
		for _, item := range sm.items {
			tmpl.Structure = append(tmpl.Structure, templates.TemplateNode{
				Name: item,
				Type: "file",
			})
		}
		return templates.SaveTemplate(tmpl)
	}
	return nil
}
