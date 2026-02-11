package main

import (
	"fmt"

	"github.com/alonsoF100/reporting-service/internal/config"
)

func main() {
	// TODO добавить инициализацию зависимостей
	cfg := config.Load()
	fmt.Print(cfg)
}
