package test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/models"
	"github.com/alonsoF100/reporting-service/internal/parser"
	"github.com/alonsoF100/reporting-service/internal/repository/postgres"
	"github.com/alonsoF100/reporting-service/internal/service"
	"github.com/alonsoF100/reporting-service/internal/transport/handler"
	"github.com/alonsoF100/reporting-service/internal/transport/router"
	"github.com/alonsoF100/reporting-service/internal/transport/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// 1. Конфиг для тестов
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Name:     "reporting-service",
			SSLMode:  "disable",
		},
		Migration: config.MigrationsConfig{
			Dir: "", // без миграций
		},
		Application: config.ApplicationConfig{
			Input:      "testdata/input",
			Output:     "testdata/output",
			Period:     1 * time.Second,
			QueueSize:  10,
			Workers:    2,
			MaxRetries: 1,
		},
		Server: config.ServerConfig{
			Port: 8081,
		},
	}

	// 2. Создаем тестовые папки
	err := os.MkdirAll(cfg.Application.Input, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(cfg.Application.Output, 0755)
	require.NoError(t, err)
	defer os.RemoveAll("testdata")

	// 3. Подключаемся к БД
	pool, err := postgres.NewPool(cfg)
	require.NoError(t, err)
	defer pool.Close()

	repo := postgres.New(pool)

	// 4. Очищаем таблицы
	_, err = pool.Exec(context.Background(), "TRUNCATE device_messages CASCADE")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), "TRUNCATE processed_files CASCADE")
	require.NoError(t, err)

	defer func() {
		_, _ = pool.Exec(context.Background(), "TRUNCATE device_messages CASCADE")
		_, _ = pool.Exec(context.Background(), "TRUNCATE processed_files CASCADE")
	}()

	// 5. Создаем тестовый TSV файл
	testFile := filepath.Join(cfg.Application.Input, "test.tsv")
	testData := `#номер	mqtt	инвентарный	гуид	id сообщения	текст сообщения	среда	классс сообщения	уровень сообщения	Зона переменных	адрес переменной
n 	mqtt	invid   	unit_guid                           	msg_id                   	text               	context	class  	level	area 	addr                                 
1 	    	G-044322	01749246-95f6-57db-b7c3-2ae0e8be671f	cold7_Defrost_status     	Разморозка         	       	waiting	100  	LOCAL	cold7_status.Defrost_status
2 	    	G-044322	01749246-95f6-57db-b7c3-2ae0e8be671f	cold7_VentSK_status      	Вентилятор         	       	working	100  	LOCAL	cold7_status.VentSK_status
3 	    	G-044325	01749246-9617-585e-9e19-157ccad61ee2	cold78_Defrost_status    	Разморозка         	       	waiting	100  	LOCAL	cold78_status.Defrost_status`

	err = os.WriteFile(testFile, []byte(testData), 0644)
	require.NoError(t, err)

	// 6. Создаем сервисы
	deviceService := service.NewDeviceService(repo)
	scanner := service.NewScanner(cfg, repo)

	// 7. Запускаем сканер в фоне
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go scanner.Start(ctx)

	// 8. Даем время на обработку
	time.Sleep(3 * time.Second)

	// 9. Проверяем, что файл обработан
	processed, err := repo.IsFileProcessed(ctx, "test.tsv")
	require.NoError(t, err)
	assert.True(t, processed, "file should be marked as processed")

	// 10. Проверяем, что сообщения сохранились в БД
	messages, err := repo.GetAllMessagesByUnitGUID(ctx, "01749246-95f6-57db-b7c3-2ae0e8be671f")
	require.NoError(t, err)
	assert.Len(t, messages, 2, "device1 should have 2 messages")

	messages, err = repo.GetAllMessagesByUnitGUID(ctx, "01749246-9617-585e-9e19-157ccad61ee2")
	require.NoError(t, err)
	assert.Len(t, messages, 1, "device2 should have 1 message")

	// 11. НЕ проверяем PDF - пропускаем
	// files, err := filepath.Glob(filepath.Join(cfg.Application.Output, "*.pdf"))
	// require.NoError(t, err)
	// assert.Len(t, files, 2, "should have 2 PDF files")

	// 12. Тестируем API
	logger := slog.Default()
	h := handler.New(deviceService)
	r := router.New(h).Setup()
	srv := server.New(cfg, h, logger)
	srv.Server.Handler = r

	go func() {
		srv.Server.Addr = ":8081"
		if err := srv.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()
	defer srv.Server.Shutdown(ctx)

	time.Sleep(1 * time.Second)

	// 13. Тестируем GET /api/v1/devices/{id}
	resp, err := http.Get("http://localhost:8081/api/v1/devices/01749246-95f6-57db-b7c3-2ae0e8be671f?page=1&limit=10")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		UnitGUID string                 `json:"unit_guid"`
		Invid    string                 `json:"invid"`
		Total    int                    `json:"total"`
		Page     int                    `json:"page"`
		Limit    int                    `json:"limit"`
		Pages    int                    `json:"pages"`
		Messages []models.DeviceMessage `json:"messages"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "01749246-95f6-57db-b7c3-2ae0e8be671f", result.UnitGUID)
	assert.Equal(t, "G-044322", result.Invid)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 1, result.Pages)
	assert.Len(t, result.Messages, 2)

	// 14. Тестируем 404
	resp, err = http.Get("http://localhost:8081/api/v1/devices/11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestParserIntegration(t *testing.T) {
	// Создаем временный TSV файл
	tmpFile, err := os.CreateTemp("", "test*.tsv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := `#номер	mqtt	инвентарный	гуид	id сообщения	текст сообщения	среда	классс сообщения	уровень сообщения	Зона переменных	адрес переменной
n 	mqtt	invid   	unit_guid                           	msg_id                   	text               	context	class  	level	area 	addr                                 
1 	    	G-044322	01749246-95f6-57db-b7c3-2ae0e8be671f	cold7_Defrost_status     	Разморозка         	       	waiting	100  	LOCAL	cold7_status.Defrost_status
2 	    	G-044322	01749246-95f6-57db-b7c3-2ae0e8be671f	cold7_VentSK_status      	Вентилятор         	       	working	100  	LOCAL	cold7_status.VentSK_status
3 	    	G-044322	01749246-95f6-57db-b7c3-2ae0e8be671f	                          		               	       		0  	      	                         
4 	    	G-044325	01749246-9617-585e-9e19-157ccad61ee2	cold78_Defrost_status    	Разморозка         	       	waiting	100  	LOCAL	cold78_status.Defrost_status`

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	// Тестируем парсер
	result, err := parser.ParseTSV(tmpFile.Name())
	require.NoError(t, err)

	// Должно быть 3 сообщения (строка 3 - пустая, пропускается)
	assert.Len(t, result.Messages, 3)

	// Проверяем первое сообщение
	assert.Equal(t, 1, result.Messages[0].Number)
	assert.Equal(t, "G-044322", result.Messages[0].Invid)
	assert.Equal(t, "01749246-95f6-57db-b7c3-2ae0e8be671f", result.Messages[0].UnitGUID)
	assert.Equal(t, "cold7_Defrost_status", result.Messages[0].MessageID)
	assert.Equal(t, "Разморозка", result.Messages[0].MessageText)
	assert.Equal(t, "waiting", result.Messages[0].MessageClass)
	assert.Equal(t, 100, result.Messages[0].Level)
	assert.Equal(t, "LOCAL", result.Messages[0].Area)
	assert.Equal(t, "cold7_status.Defrost_status", result.Messages[0].Address)
}
