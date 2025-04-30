package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template은 폴더 구조 템플릿을 나타냅니다
type Template struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Variables   []string       `json:"variables"`
	Structure   []TemplateNode `json:"structure"`
}

// TemplateNode는 템플릿의 각 노드(폴더/파일)를 나타냅니다
type TemplateNode struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"` // "dir" 또는 "file"
	Children []TemplateNode `json:"children,omitempty"`
}

// TemplateManager는 템플릿을 관리하는 인터페이스입니다
type TemplateManager interface {
	Save(template Template) error
	Load(name string) (*Template, error)
	List() ([]Template, error)
	Delete(name string) error
	Apply(template *Template, path string, variables map[string]string) error
}

// FileTemplateManager는 파일 시스템 기반의 템플릿 관리자입니다
type FileTemplateManager struct {
	baseDir string
}

// NewFileTemplateManager는 새로운 FileTemplateManager를 생성합니다
func NewFileTemplateManager(baseDir string) (*FileTemplateManager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FileTemplateManager{baseDir: baseDir}, nil
}

// Save는 템플릿을 파일로 저장합니다
func (m *FileTemplateManager) Save(template Template) error {
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.baseDir, template.Name+".json"), data, 0644)
}

// Load는 템플릿을 파일에서 로드합니다
func (m *FileTemplateManager) Load(name string) (*Template, error) {
	data, err := os.ReadFile(filepath.Join(m.baseDir, name+".json"))
	if err != nil {
		return nil, err
	}
	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

// List는 저장된 모든 템플릿을 반환합니다
func (m *FileTemplateManager) List() ([]Template, error) {
	files, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}

	var templates []Template
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			template, err := m.Load(file.Name()[:len(file.Name())-5])
			if err != nil {
				continue
			}
			templates = append(templates, *template)
		}
	}
	return templates, nil
}

// Delete는 템플릿을 삭제합니다
func (m *FileTemplateManager) Delete(name string) error {
	return os.Remove(filepath.Join(m.baseDir, name+".json"))
}

// Apply는 템플릿을 지정된 경로에 적용합니다
func (m *FileTemplateManager) Apply(template *Template, path string, variables map[string]string) error {
	// 변수 검증
	for _, v := range template.Variables {
		if _, ok := variables[v]; !ok {
			return fmt.Errorf("필수 변수 '%s'가 제공되지 않았습니다", v)
		}
	}

	// 루트 디렉토리 생성
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("루트 디렉토리를 생성할 수 없습니다: %v", err)
	}

	// 각 노드에 대해 재귀적으로 처리
	for _, node := range template.Structure {
		if err := m.applyNode(node, path, variables); err != nil {
			return err
		}
	}

	return nil
}

// applyNode는 단일 노드를 처리합니다
func (m *FileTemplateManager) applyNode(node TemplateNode, basePath string, variables map[string]string) error {
	// 변수 치환
	name := node.Name
	for k, v := range variables {
		name = strings.ReplaceAll(name, "{"+k+"}", v)
	}

	path := filepath.Join(basePath, name)

	switch node.Type {
	case "dir":
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("디렉토리를 생성할 수 없습니다 '%s': %v", path, err)
		}
		// 하위 노드 처리
		for _, child := range node.Children {
			if err := m.applyNode(child, path, variables); err != nil {
				return err
			}
		}
	case "file":
		// 상위 디렉토리 생성
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("상위 디렉토리를 생성할 수 없습니다 '%s': %v", dir, err)
		}
		// 빈 파일 생성
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			return fmt.Errorf("파일을 생성할 수 없습니다 '%s': %v", path, err)
		}
	default:
		return fmt.Errorf("알 수 없는 노드 타입: %s", node.Type)
	}

	return nil
}

// ScanDirectoryRecursive는 지정된 경로를 재귀적으로 스캔하여 TemplateNode 슬라이스를 반환합니다.
// maxDepth가 0이 아니면 해당 깊이까지만 스캔합니다.
// .git 디렉토리와 .DS_Store 파일은 무시합니다.
func ScanDirectoryRecursive(targetPath string, currentDepth int, maxDepth int) ([]TemplateNode, error) {
	// 최대 깊이 도달 시 빈 슬라이스 반환 (에러 아님)
	if maxDepth > 0 && currentDepth >= maxDepth {
		return []TemplateNode{}, nil
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, fmt.Errorf("디렉토리를 읽을 수 없습니다 '%s': %w", targetPath, err)
	}

	var nodes []TemplateNode
	ignoredNames := map[string]bool{".git": true, ".DS_Store": true}

	for _, entry := range entries {
		name := entry.Name()
		if ignoredNames[name] {
			continue // 무시 목록에 있으면 건너뜀
		}

		fullPath := filepath.Join(targetPath, name)
		node := TemplateNode{Name: name}

		if entry.IsDir() {
			node.Type = "dir"
			// 다음 깊이로 재귀 호출
			children, err := ScanDirectoryRecursive(fullPath, currentDepth+1, maxDepth)
			if err != nil {
				return nil, err // 하위 디렉토리 스캔 오류 시 중단
			}
			node.Children = children
		} else {
			node.Type = "file"
		}
		nodes = append(nodes, node)
	}

	// 파일/디렉토리 정렬 (이름 순, 디렉토리 우선)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type == "dir"
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

// SaveTemplate은 템플릿을 파일로 저장합니다
func SaveTemplate(template *Template) error {
	// 템플릿 디렉토리 생성
	templateDir := filepath.Join(os.Getenv("HOME"), ".tree-generator", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("템플릿 디렉토리 생성 실패: %v", err)
	}

	// 템플릿 파일 경로
	templatePath := filepath.Join(templateDir, template.Name+".json")

	// 템플릿을 JSON으로 변환
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("템플릿 JSON 변환 실패: %v", err)
	}

	// 파일로 저장
	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		return fmt.Errorf("템플릿 저장 실패: %v", err)
	}

	return nil
}
