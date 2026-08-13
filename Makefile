# issue2md Makefile — 标准化操作入口

.PHONY: test build run clean

## 运行所有测试
test:
	go test ./...

## 构建 CLI 二进制
build:
	go build -o issue2md ./cmd/issue2md

## 本地运行（示例：issue2md <url>）
run:
	go run ./cmd/issue2md $(ARGS)

## 清理构建产物
clean:
	rm -f issue2md issue2md.exe
