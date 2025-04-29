# tree-generator

폴더/파일 구조 템플릿을 CLI로 만들고 적용할 수 있는 도구입니다.

## 주요 기능

- 템플릿 생성(TUI): 원하는 폴더/파일 구조를 간단하게 입력하여 템플릿으로 저장
- 템플릿 적용: 저장된 템플릿을 원하는 경로에 실제 폴더/파일로 생성
- 템플릿 리스트: 저장된 템플릿 목록 확인

## 설치

```bash
go build -o tg cmd/tg/main.go
```

## 사용법

### 1. 템플릿 생성

```bash
./tg create
```

- 순서대로 템플릿 이름, 설명, 폴더/파일명을 입력합니다.
- 폴더/파일명은 여러 개 입력 가능하며, 빈 입력 후 Enter를 누르면 종료됩니다.
- 템플릿은 `~/.tree-generator/templates/`에 `<이름>.json` 파일로 저장됩니다.

### 2. 템플릿 목록 확인

```bash
./tg list
```

### 3. 템플릿 적용

```bash
./tg apply -n <템플릿이름> -p <적용할_경로>
```

- 예시: `./tg apply -n hi -p .`
- 템플릿에 정의된 폴더/파일 구조가 지정한 경로에 생성됩니다.

## 템플릿 저장 위치

- 모든 템플릿은 `~/.tree-generator/templates/` 폴더에 JSON 파일로 저장됩니다.

## 예시

```
$ ./tg create
[템플릿 이름]
my-template
[템플릿 설명]
간단한 예시
[폴더/파일 이름 입력] (빈 입력시 종료)
foo.txt
bar/baz.txt
<빈 입력 후 Enter>
저장 완료!

$ ./tg list
my-template

$ ./tg apply -n my-template -p ./test
```

## 라이선스

MIT
