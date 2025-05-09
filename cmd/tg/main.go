package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wdwb/tree-generator/internal/templates"
	"github.com/wdwb/tree-generator/internal/tui"
)

var (
	templateManager templates.TemplateManager
	configFilePath  string
)

// Config 구조체는 애플리케이션 설정을 나타냅니다.
type Config struct {
	DefaultTemplate string `json:"default_template"`
}

// loadConfig는 설정 파일에서 설정을 로드합니다.
func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 설정 파일이 없으면 기본값 반환
			return &Config{}, nil
		}
		return nil, fmt.Errorf("설정 파일을 읽을 수 없습니다: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("설정 파일 파싱 오류: %w", err)
	}
	return &config, nil
}

// saveConfig는 설정을 파일에 저장합니다.
func saveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("설정 JSON 변환 실패: %w", err)
	}
	// 설정 디렉토리가 없으면 생성
	if err := os.MkdirAll(filepath.Dir(configFilePath), 0755); err != nil {
		return fmt.Errorf("설정 디렉토리 생성 실패: %w", err)
	}
	return os.WriteFile(configFilePath, data, 0644)
}

func init() {
	// 템플릿 및 설정 저장 디렉토리 설정
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("홈 디렉토리를 찾을 수 없습니다: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Join(homeDir, ".tree-generator")
	templateDir := filepath.Join(baseDir, "templates")
	configFilePath = filepath.Join(baseDir, "config.json") // 설정 파일 경로 정의

	// 템플릿 관리자 초기화
	templateManager, err = templates.NewFileTemplateManager(templateDir)
	if err != nil {
		fmt.Printf("템플릿 관리자를 초기화할 수 없습니다: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "Tree Generator - 폴더 구조 생성 도구",
	Long: `Tree Generator는 폴더 구조를 쉽게 생성하고 관리할 수 있는 도구입니다.
템플릿을 저장하고 재사용할 수 있으며, 변수를 사용하여 동적인 폴더 구조를 만들 수 있습니다.`,
}

// applyTemplate 함수: 템플릿 적용 로직 분리
func applyTemplate(templateName string, targetPath string) error {
	template, err := templateManager.Load(templateName)
	if err != nil {
		return fmt.Errorf("템플릿을 로드할 수 없습니다: %w", err)
	}

	// 변수 입력 받기
	variables := make(map[string]string)
	if len(template.Variables) > 0 {
		fmt.Println("템플릿 변수 값을 입력하세요:")
		for _, v := range template.Variables {
			fmt.Printf("%s: ", v)
			var value string
			// TODO: Read input more robustly if needed
			fmt.Scanln(&value)
			variables[v] = value
		}
	}

	fmt.Printf("템플릿 '%s'를 '%s' 경로에 적용합니다.\n", templateName, targetPath)
	if err := templateManager.Apply(template, targetPath, variables); err != nil {
		return fmt.Errorf("템플릿 적용 중 오류가 발생했습니다: %w", err)
	}
	fmt.Println("템플릿이 성공적으로 적용되었습니다.")
	return nil
}

func init() {
	// apply 명령어
	applyCmd := &cobra.Command{
		Use:   "apply [template_name]",
		Short: "저장된 템플릿을 적용하여 폴더 구조 생성",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var templateName string
			path, _ := cmd.Flags().GetString("path")

			if len(args) == 1 {
				// 인자가 있으면 해당 이름 사용
				templateName = args[0]
				// TODO: 인자로 받은 템플릿 이름이 실제 존재하는지 확인하는 로직 추가하면 더 좋음
			} else {
				// 인자가 없으면 설정 파일에서 기본 템플릿 로드
				config, err := loadConfig()
				if err != nil {
					fmt.Printf("설정 로드 오류: %v\n", err)
					return
				}
				if config.DefaultTemplate == "" {
					fmt.Println("적용할 템플릿이 지정되지 않았습니다.")
					fmt.Printf("사용법: %s apply <template_name> 또는 %s use <template_name>으로 기본값 설정\n", os.Args[0], os.Args[0])
					return
				}
				templateName = config.DefaultTemplate // 기본 템플릿 사용
				fmt.Printf("기본 템플릿 '%s'를 사용합니다.\n", templateName)
			}

			if err := applyTemplate(templateName, path); err != nil {
				fmt.Printf("%v\n", err)
				return
			}
		},
	}
	applyCmd.Flags().StringP("path", "p", ".", "적용할 경로")

	// create 명령어
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "새로운 폴더 구조 템플릿 생성",
		Run: func(cmd *cobra.Command, args []string) {
			if err := tui.StartTUI(); err != nil {
				fmt.Printf("TUI 실행 중 오류가 발생했습니다: %v\n", err)
				return
			}
		},
	}

	listCmd := &cobra.Command{
		Use:   "list [template_name]",
		Short: "저장된 템플릿의 구조를 트리 형태로 출력합니다",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var selectedTemplateName string
			var err error

			if len(args) == 1 {
				// 인자가 있으면 해당 이름 사용
				selectedTemplateName = args[0]
			} else {
				// 인자가 없으면 TUI 실행
				templatesList, err := templateManager.List()
				if err != nil {
					fmt.Printf("템플릿 목록을 가져올 수 없습니다: %v\n", err)
					return
				}
				if len(templatesList) == 0 {
					fmt.Println("저장된 템플릿이 없습니다.")
					return
				}

				// 현재 기본값 로드 (TUI 표시용)
				currentConfig, err := loadConfig()
				if err != nil {
					fmt.Printf("경고: 설정을 로드하는 중 오류 발생: %v\n", err)
					currentConfig = &Config{} // 빈 설정으로 진행
				}

				selectedTemplateName, err = tui.SelectTemplateTUI(templatesList, currentConfig.DefaultTemplate) // TUI 호출
				if err != nil {
					fmt.Printf("TUI 실행 중 오류가 발생했습니다: %v\n", err)
					return
				}
				if selectedTemplateName == "" { // 사용자가 TUI에서 취소한 경우
					// 메시지는 TUI에서 출력하므로 바로 종료
					return
				}
			}

			// 선택된 템플릿 로드 및 출력
			tmpl, err := templateManager.Load(selectedTemplateName)
			if err != nil {
				// TUI나 인자에서 검증된 이름이므로 Load 실패는 예상치 못한 오류
				// 다만, TUI 실행과 Load 사이에 파일이 삭제된 경우 발생 가능
				fmt.Printf("템플릿 '%s'를 불러올 수 없습니다: %v\n", selectedTemplateName, err)
				return
			}
			fmt.Printf("Template: %s (%s)\n", tmpl.Name, tmpl.Description)
			fmt.Printf("--------------Tree------------------\n")
			printTree(tmpl.Structure, "")
		},
	}

	// use 명령어 추가
	useCmd := &cobra.Command{
		Use:   "use [template_name]",
		Short: "TUI를 통해 템플릿을 선택하거나 인자로 전달된 템플릿을 사용합니다",
		Args:  cobra.MaximumNArgs(1), // 최대 1개의 인자만 허용
		Run: func(cmd *cobra.Command, args []string) {
			var selectedTemplateName string
			var err error

			templatesList, err := templateManager.List()
			if err != nil {
				fmt.Printf("템플릿 목록을 가져올 수 없습니다: %v\n", err)
				return
			}

			if len(templatesList) == 0 {
				fmt.Println("저장된 템플릿이 없습니다.")
				return
			}

			if len(args) == 1 {
				// 인자가 있으면 해당 템플릿 이름 사용
				selectedTemplateName = args[0]
				// 해당 이름의 템플릿이 존재하는지 확인
				exists := false
				for _, t := range templatesList {
					if t.Name == selectedTemplateName {
						exists = true
						break
					}
				}
				if !exists {
					fmt.Printf("템플릿 '%s'를 찾을 수 없습니다.\n", selectedTemplateName)
					return
				}
			} else {
				// 인자가 없으면 TUI 실행
				// TUI 호출 전에 현재 설정을 로드
				currentConfig, err := loadConfig()
				if err != nil {
					// 설정 로드 오류는 치명적이지 않게 처리하고 TUI는 계속 진행
					fmt.Printf("경고: 설정을 로드하는 중 오류 발생: %v\n", err)
					currentConfig = &Config{} // 빈 설정으로 진행
				}

				selectedTemplateName, err = tui.SelectTemplateTUI(templatesList, currentConfig.DefaultTemplate) // 현재 기본값 전달
				if err != nil {
					fmt.Printf("TUI 실행 중 오류가 발생했습니다: %v\n", err)
					return
				}
				if selectedTemplateName == "" { // 사용자가 TUI에서 취소한 경우
					// 메시지는 TUI 내부에서 출력하므로 여기서는 바로 종료
					return
				}
			}

			// 기본 템플릿으로 설정 저장
			config, err := loadConfig()
			if err != nil {
				fmt.Printf("설정을 로드하는 중 오류 발생: %v\n", err)
				return
			}
			config.DefaultTemplate = selectedTemplateName
			if err := saveConfig(config); err != nil {
				fmt.Printf("설정을 저장하는 중 오류 발생: %v\n", err)
				return
			}

			fmt.Printf("기본 템플릿이 '%s'(으)로 설정되었습니다. '%s apply'를 사용하여 적용하세요.\n", selectedTemplateName, os.Args[0])

		},
	}

	// clone 명령어 추가
	cloneCmd := &cobra.Command{
		Use:   "clone <path> <template_name> <description>",
		Short: "지정된 경로의 디렉토리 구조를 스캔하여 새 템플릿으로 저장합니다",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			// 인자 파싱
			path := args[0]
			templateName := args[1]
			description := args[2]
			maxDepth, _ := cmd.Flags().GetInt("depth") // depth 플래그 값 읽기

			fmt.Printf("'%s' 경로의 구조를 최대 깊이 %d까지 스캔하여 '%s' 템플릿으로 저장합니다 (설명: %s)...\n", path, maxDepth, templateName, description)
			if maxDepth == 0 {
				fmt.Printf("'%s' 경로의 구조를 스캔하여 '%s' 템플릿으로 저장합니다 (설명: %s)...\n", path, templateName, description)
			} else {
				fmt.Printf("'%s' 경로의 구조를 최대 깊이 %d까지 스캔하여 '%s' 템플릿으로 저장합니다 (설명: %s)...\n", path, maxDepth, templateName, description)
			}

			// 1. 경로 스캔 (depth 정보 전달 필요)
			// structure, err := templates.ScanDirectory(path)
			structure, err := templates.ScanDirectoryRecursive(path, 0, maxDepth) // 수정된 함수 호출 (가칭)
			if err != nil {
				fmt.Printf("경로 스캔 중 오류 발생: %v\n", err)
				return
			}

			// 2. Template 구조체 생성
			template := templates.Template{
				Name:        templateName,
				Description: description,
				Variables:   nil,
				Structure:   structure,
			}

			// 3. 템플릿 저장
			if err := templateManager.Save(template); err != nil {
				fmt.Printf("템플릿 저장 중 오류 발생: %v\n", err)
				return
			}

			fmt.Printf("템플릿 '%s'가 성공적으로 저장되었습니다.\n", templateName)
		},
	}
	cloneCmd.Flags().IntP("depth", "d", 0, "스캔할 최대 디렉토리 깊이 (0은 무제한)") // depth 플래그 추가

	// remove 명령어 추가
	removeCmd := &cobra.Command{
		Use:   "remove [template_name...]",
		Short: "저장된 템플릿을 삭제합니다",
		Long:  "인자 없이 실행하면 TUI를 통해 삭제할 템플릿을 선택할 수 있습니다.",
		Run: func(cmd *cobra.Command, args []string) {
			var templatesToDelete []string

			if len(args) > 0 {
				// 인자가 있으면 해당 템플릿들을 삭제 목록에 추가
				templatesToDelete = args
			} else {
				// 인자가 없으면 TUI 실행
				templatesList, err := templateManager.List()
				if err != nil {
					fmt.Printf("템플릿 목록을 가져올 수 없습니다: %v\n", err)
					return
				}
				if len(templatesList) == 0 {
					fmt.Println("삭제할 템플릿이 없습니다.")
					return
				}

				// TUI를 호출하여 삭제할 템플릿 목록을 받음
				templatesToDelete, err = tui.SelectTemplatesToDeleteTUI(templatesList)
				if err != nil {
					fmt.Printf("TUI 실행 중 오류가 발생했습니다: %v\n", err)
					return
				}
			}

			if len(templatesToDelete) == 0 {
				fmt.Println("삭제할 템플릿이 선택되지 않았습니다.")
				return
			}

			fmt.Printf("다음 템플릿을 삭제하시겠습니까? %v\n", templatesToDelete)
			fmt.Print("진행하려면 'yes'를 입력하세요: ")
			var confirm string
			fmt.Scanln(&confirm)

			if confirm != "yes" {
				fmt.Println("삭제가 취소되었습니다.")
				return
			}

			// 선택된 템플릿 삭제
			deletedCount := 0
			failedCount := 0
			for _, name := range templatesToDelete {
				if err := templateManager.Delete(name); err != nil {
					fmt.Printf("템플릿 '%s' 삭제 실패: %v\n", name, err)
					failedCount++
				} else {
					fmt.Printf("템플릿 '%s' 삭제 완료\n", name)
					deletedCount++
				}
			}
			fmt.Printf("총 %d개 템플릿 삭제 완료, %d개 실패\n", deletedCount, failedCount)
		},
	}

	rootCmd.AddCommand(applyCmd, createCmd, listCmd, useCmd, cloneCmd, removeCmd)
}

func printTree(nodes []templates.TemplateNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// 현재 노드의 연결선 결정
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		// 현재 노드 출력
		fmt.Printf("%s%s%s\n", prefix, connector, node.Name)

		// 자식 노드를 위한 접두사 준비
		childPrefix := prefix
		if isLast {
			// 현재 노드가 마지막이면, 자식 노드는 세로선을 그리지 않음
			childPrefix += "    "
		} else {
			// 현재 노드가 마지막이 아니면, 자식 노드는 세로선을 그림
			childPrefix += "│   "
		}

		// 디렉토리인 경우 자식 노드 재귀 호출
		if node.Type == "dir" && len(node.Children) > 0 {
			printTree(node.Children, childPrefix)
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
