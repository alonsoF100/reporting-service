package main

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/logger"
	"github.com/alonsoF100/reporting-service/internal/parser"
)

func main() {
	// TODO добавить инициализацию зависимостей
	cfg := config.Load()
	log := logger.Setup(cfg)
	slog.SetDefault(log)

	fmt.Println("\n=== ТЕСТИРОВАНИЕ ПАРСЕРА ===")

	testFile := filepath.Join(cfg.Application.Input, "data.tsv")

	result, err := parser.ParseTSV(testFile)
	if err != nil {
		slog.Error("Ошибка парсинга", "error", err)
		return
	}

	fmt.Printf("Файл: %s\n", result.FileName)
	fmt.Printf("Сообщений: %d\n\n", len(result.Messages))

	for i, msg := range result.Messages {
		if i >= 3 {
			break
		}
		fmt.Printf("%d. GUID: %s\n", i+1, msg.UnitGUID)
		fmt.Printf("   Текст: %s\n", msg.MessageText)
		fmt.Printf("   Класс: %s\n\n", msg.MessageClass)
	}

	fmt.Println("=== ТЕСТ ЗАВЕРШЕН ===")
}
