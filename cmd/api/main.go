package main

import (
	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/logger"
)

func main() {
	// TODO добавить инициализацию зависимостей
	cfg := config.Load()

	_ = logger.Setup(cfg)
}
