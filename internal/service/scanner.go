package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/alonsoF100/reporting-service/internal/config"
	"github.com/alonsoF100/reporting-service/internal/models"
	"github.com/alonsoF100/reporting-service/internal/parser"
	"github.com/jung-kurt/gofpdf"
)

type Repository interface {
	// Проверка, обработан ли файл
	IsFileProcessed(ctx context.Context, fileName string) (bool, error)

	// Получить все обработанные файлы
	GetAllProcessedFiles(ctx context.Context) ([]models.ProcessedFile, error)

	// Обновляет статус файла и сообщение об ошибке
	UpdateFileStatus(ctx context.Context, fileName string, status, errorMsg string) error

	// Сохранить сообщения из файла
	SaveMessages(ctx context.Context, messages []models.DeviceMessage) error

	// Получить все сообщения устройства
	GetAllMessagesByUnitGUID(ctx context.Context, unitGUID string) ([]models.DeviceMessage, error)
}

type Scanner struct {
	cfg    *config.Config
	repo   Repository
	queue  chan string
	logger *slog.Logger
}

func NewScanner(cfg *config.Config, repo Repository) *Scanner {
	queue := make(chan string, cfg.Application.QueueSize)

	return &Scanner{
		cfg:    cfg,
		repo:   repo,
		queue:  queue,
		logger: slog.With("component", "scanner"),
	}
}

// Start запускает периодическое сканирование
func (s *Scanner) Start(ctx context.Context) {
	for i := 0; i < s.cfg.Application.Workers; i++ {
		go s.Worker(ctx, i)
	}
	s.logger.Info("workers started", "count", s.cfg.Application.Workers)

	ticker := time.NewTicker(s.cfg.Application.Period)
	defer ticker.Stop()

	s.logger.Info("scanner started",
		"interval", s.cfg.Application.Period,
		"queue_size", s.cfg.Application.QueueSize)

	s.Scan(ctx)

	for {
		select {
		case <-ticker.C:
			s.Scan(ctx)
		case <-ctx.Done():
			s.logger.Info("scanner stopped")
			return
		}
	}
}

// Scan - основная логика сканирования
func (s *Scanner) Scan(ctx context.Context) {
	s.logger.Info("scanning directory", "dir", s.cfg.Application.Input)

	dbFiles, err := s.repo.GetAllProcessedFiles(ctx)
	if err != nil {
		s.logger.Error("failed to get processed files from DB", "error", err)
		return
	}

	processedMap := make(map[string]string)
	for _, f := range dbFiles {
		processedMap[f.FileName] = f.Status
	}

	dirEntries, err := os.ReadDir(s.cfg.Application.Input)
	if err != nil {
		s.logger.Error("failed to read directory", "error", err)
		return
	}

	newFiles := []string{}
	for _, entry := range dirEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".tsv" {
			continue
		}

		fileName := entry.Name()
		status, exists := processedMap[fileName]

		if !exists {
			newFiles = append(newFiles, fileName)
			s.logger.Info("new file found", "file", fileName)
			continue
		}

		if status == models.StatusError {
			newFiles = append(newFiles, fileName)
			s.logger.Info("retry file with error", "file", fileName)
		}
	}

	for _, fileName := range newFiles {
		fullPath := filepath.Join(s.cfg.Application.Input, fileName)

		select {
		case s.queue <- fullPath:
			s.logger.Info("file added to queue", "file", fileName)
		default:
			s.logger.Error("queue is full, skipping file",
				"file", fileName,
				"queue_size", s.cfg.Application.QueueSize)
		}
	}

	s.logger.Info("scan completed",
		"new_files", len(newFiles),
		"queue_size", len(s.queue))
}

// Worker обрабатывает файлы из очереди
func (s *Scanner) Worker(ctx context.Context, id int) {
	s.logger.Info("worker started", "worker_id", id)

	for {
		select {
		case filePath := <-s.queue:
			fileName := filepath.Base(filePath)

			retryCount := 0
			maxRetries := s.cfg.Application.MaxRetries

			err := s.repo.UpdateFileStatus(ctx, fileName, models.StatusProcessing, "")
			if err != nil {
				s.logger.Error("failed to mark file as processing",
					"worker_id", id,
					"file", fileName,
					"error", err)
			}

			// обрабатываем файл + механизм попыток
			for retryCount < maxRetries {
				err = s.processFile(ctx, filePath, fileName)
				if err == nil {
					s.repo.UpdateFileStatus(ctx, fileName, models.StatusProcessed, "")
					s.logger.Info("file processed successfully",
						"worker_id", id,
						"file", fileName,
						"attempt", retryCount+1)
					break
				}

				retryCount++
				s.logger.Error("failed to process file",
					"worker_id", id,
					"file", fileName,
					"attempt", retryCount,
					"max_retries", maxRetries,
					"error", err)

				if retryCount < maxRetries {
					waitTime := time.Duration(retryCount*2) * time.Second
					s.logger.Info("retrying file",
						"worker_id", id,
						"file", fileName,
						"wait_time", waitTime,
						"next_attempt", retryCount+1)
					time.Sleep(waitTime)
				}
			}

			// Если не сумели за n попыток, то помечаем ошибкой
			if retryCount == maxRetries && err != nil {
				s.repo.UpdateFileStatus(ctx, fileName, models.StatusError, err.Error())
				s.logger.Error("file failed after all retries",
					"worker_id", id,
					"file", fileName,
					"max_retries", maxRetries,
					"error", err)
			}

		case <-ctx.Done():
			s.logger.Info("worker stopped", "worker_id", id)
			return
		}
	}
}

// processFile - основная логика обработки файла
func (s *Scanner) processFile(ctx context.Context, filePath, fileName string) error {
	s.logger.Info("processing file", "file", fileName)

	// парсим файл
	parseResult, err := parser.ParseTSV(filePath)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if len(parseResult.Messages) == 0 {
		return fmt.Errorf("no messages found in file")
	}

	s.logger.Info("file parsed successfully",
		"file", fileName,
		"messages", len(parseResult.Messages))

	// сейвим в базу сообщения
	err = s.repo.SaveMessages(ctx, parseResult.Messages)
	if err != nil {
		return fmt.Errorf("save messages error: %w", err)
	}

	s.logger.Info("messages saved to DB",
		"file", fileName,
		"messages", len(parseResult.Messages))

	// получаем уникальные девайсы
	uniqueDevices := make(map[string]bool)
	for _, msg := range parseResult.Messages {
		uniqueDevices[msg.UnitGUID] = true
	}

	// для каждого девайса уникального генерим пдфку
	for unitGUID := range uniqueDevices {
		messages, err := s.repo.GetAllMessagesByUnitGUID(ctx, unitGUID)
		if err != nil {
			s.logger.Error("failed to get messages for device",
				"unit_guid", unitGUID,
				"error", err)
			continue
		}

		// Генерим пдфку
		outputPath := filepath.Join(s.cfg.Application.Output, fmt.Sprintf("%s.pdf", unitGUID))
		err = s.generatePDF(unitGUID, messages, outputPath)
		if err != nil {
			s.logger.Error("failed to generate PDF",
				"unit_guid", unitGUID,
				"error", err)
			continue
		}

		s.logger.Info("PDF generated/updated",
			"unit_guid", unitGUID,
			"messages", len(messages),
			"path", outputPath)
	}

	// по идее можно файл обработанный убрать из input папки и кинуть, допустим в архив или что-то такое
	// в тз нету, поэтому оставляю так

	return nil
}

// generatePDF - генерирует реальный PDF с данными устройства
func (s *Scanner) generatePDF(unitGUID string, messages []models.DeviceMessage, outputPath string) error {
	// Создаем PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Добавляем шрифт с поддержкой кириллицы (DejaVu)
	pdf.AddUTF8Font("DejaVu", "", "fonts/DejaVuSans.ttf")
	pdf.SetFont("DejaVu", "", 12)

	// Заголовок
	pdf.SetFont("DejaVu", "B", 16)
	pdf.Cell(0, 10, "Отчет по устройству")
	pdf.Ln(15)

	// Информация об устройстве
	pdf.SetFont("DejaVu", "", 11)
	pdf.Cell(0, 7, "Unit GUID: "+unitGUID)
	pdf.Ln(8)

	if len(messages) > 0 {
		pdf.Cell(0, 7, "Инвентарный номер: "+messages[0].Invid)
		pdf.Ln(8)
	}

	pdf.Cell(0, 7, "Всего сообщений: "+fmt.Sprint(len(messages)))
	pdf.Ln(8)

	pdf.Cell(0, 7, "Дата отчета: "+time.Now().Format("02.01.2006 15:04:05"))
	pdf.Ln(15)

	// Таблица сообщений
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(15, 7, "№", "1", 0, "C", false, 0, "")
	pdf.CellFormat(70, 7, "Сообщение", "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 7, "Класс", "1", 0, "C", false, 0, "")
	pdf.CellFormat(20, 7, "Уровень", "1", 0, "C", false, 0, "")
	pdf.CellFormat(55, 7, "Адрес", "1", 0, "C", false, 0, "")
	pdf.Ln(-1)

	// Данные
	pdf.SetFont("DejaVu", "", 9)
	for i, msg := range messages {
		if i >= 30 { // Ограничим до 30 строк
			pdf.SetFont("DejaVu", "I", 9)
			pdf.CellFormat(0, 7, "... и еще "+fmt.Sprint(len(messages)-30)+" сообщений", "", 0, "L", false, 0, "")
			break
		}

		// Обрезаем длинные строки
		text := msg.MessageText
		if len(text) > 30 {
			text = text[:27] + "..."
		}

		addr := msg.Address
		if len(addr) > 25 {
			addr = addr[:22] + "..."
		}

		pdf.CellFormat(15, 7, fmt.Sprint(i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(70, 7, text, "1", 0, "L", false, 0, "")
		pdf.CellFormat(30, 7, msg.MessageClass, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprint(msg.Level), "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 7, addr, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	return pdf.OutputFileAndClose(outputPath)
}
