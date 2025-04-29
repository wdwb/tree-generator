package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/wdwb/tree-generator/internal/templates"
	"github.com/wdwb/tree-generator/internal/tui"
)

var (
	templateManager templates.TemplateManager
)

func init() {
	// 템플릿 저장 디렉토리 설정
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("홈 디렉토리를 찾을 수 없습니다: %v\n", err)
		os.Exit(1)
	}
	templateDir := filepath.Join(homeDir, ".tree-generator", "templates")

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

func init() {
	// apply 명령어
	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "저장된 템플릿을 적용하여 폴더 구조 생성",
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			path, _ := cmd.Flags().GetString("path")

			template, err := templateManager.Load(name)
			if err != nil {
				fmt.Printf("템플릿을 로드할 수 없습니다: %v\n", err)
				return
			}

			// 변수 입력 받기
			variables := make(map[string]string)
			for _, v := range template.Variables {
				fmt.Printf("%s 값을 입력하세요: ", v)
				var value string
				fmt.Scanln(&value)
				variables[v] = value
			}

			fmt.Printf("템플릿 '%s'를 '%s' 경로에 적용합니다.\n", name, path)
			if err := templateManager.Apply(template, path, variables); err != nil {
				fmt.Printf("템플릿 적용 중 오류가 발생했습니다: %v\n", err)
				return
			}
			fmt.Println("템플릿이 성공적으로 적용되었습니다.")
		},
	}
	applyCmd.Flags().StringP("name", "n", "", "템플릿 이름")
	applyCmd.Flags().StringP("path", "p", ".", "적용할 경로")
	applyCmd.MarkFlagRequired("name")

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

	// list 명령어
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "저장된 템플릿 목록 보기",
		Run: func(cmd *cobra.Command, args []string) {
			templates, err := templateManager.List()
			if err != nil {
				fmt.Printf("템플릿 목록을 가져올 수 없습니다: %v\n", err)
				return
			}

			if len(templates) == 0 {
				fmt.Println("저장된 템플릿이 없습니다.")
				return
			}

			fmt.Println("저장된 템플릿 목록:")
			for _, t := range templates {
				fmt.Printf("- %s: %s\n", t.Name, t.Description)
			}
		},
	}

	rootCmd.AddCommand(applyCmd, createCmd, listCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
