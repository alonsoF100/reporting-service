package postgres

import (
	"context"

	"github.com/alonsoF100/reporting-service/internal/models"
)

// Проверка, обработан ли файл
func (r Repository) IsFileProcessed(ctx context.Context, fileName string) (bool, error) {
	return true, nil
}

// Получить все обработанные файлы
func (r Repository) GetAllProcessedFiles(ctx context.Context) ([]models.ProcessedFile, error) {
	return nil, nil
}

// Обновляет статус файла и сообщение об ошибке
func (r Repository) UpdateFileStatus(ctx context.Context, fileName string, status, errorMsg string) error {
	return nil
}

// Сохранить сообщения из файла
func (r Repository) SaveMessages(ctx context.Context, messages []models.DeviceMessage) error {
	return nil
}

// Получить все сообщения устройства
func (r Repository) GetAllMessagesByUnitGUID(ctx context.Context, unitGUID string) ([]models.DeviceMessage, error) {
	return nil, nil
}
