package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/alonsoF100/reporting-service/internal/models"
)

// ----------------------------------------------------------------------------
// Processed files methods
// ----------------------------------------------------------------------------

// UpdateFileStatus - обновляет статус файла (processing/processed/error)
func (r *Repository) UpdateFileStatus(ctx context.Context, fileName, status, errorMsg string) error {
	const op = "postgres.UpdateFileStatus"

	logger := r.logger.With(
		slog.String("op", op),
		slog.String("file", fileName),
		slog.String("status", status),
	)

	logger.Info("updating file status")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Insert("processed_files").
		Columns("file_name", "status", "error_message", "processed_at").
		Values(fileName, status, errorMsg, time.Now()).
		Suffix("ON CONFLICT (file_name) DO UPDATE SET status = $2, error_message = $3, processed_at = $4").
		ToSql()

	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		logger.Error("failed to update file status", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("file status updated")
	return nil
}

// GetAllProcessedFiles - возвращает все обработанные файлы
func (r *Repository) GetAllProcessedFiles(ctx context.Context) ([]models.ProcessedFile, error) {
	const op = "postgres.GetAllProcessedFiles"

	logger := r.logger.With(slog.String("op", op))
	logger.Info("getting all processed files")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Select("id", "file_name", "status", "error_message", "processed_at", "created_at").
		From("processed_files").
		OrderBy("processed_at DESC").
		ToSql()

	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		logger.Error("failed to query processed files", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var files []models.ProcessedFile

	for rows.Next() {
		var f models.ProcessedFile
		err := rows.Scan(
			&f.ID,
			&f.FileName,
			&f.Status,
			&f.ErrorMessage,
			&f.ProcessedAt,
			&f.CreatedAt,
		)
		if err != nil {
			logger.Error("failed to scan row", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		files = append(files, f)
	}

	logger.Info("processed files retrieved", slog.Int("count", len(files)))
	return files, nil
}

// IsFileProcessed - проверяет, обработан ли файл
func (r *Repository) IsFileProcessed(ctx context.Context, fileName string) (bool, error) {
	const op = "postgres.IsFileProcessed"

	logger := r.logger.With(
		slog.String("op", op),
		slog.String("file", fileName),
	)

	logger.Debug("checking if file is processed")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Select("COUNT(*)").
		From("processed_files").
		Where(sq.Eq{"file_name": fileName}).
		ToSql()

	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return false, fmt.Errorf("%s: build query: %w", op, err)
	}

	var count int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		logger.Error("failed to check file status", slog.String("error", err.Error()))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	logger.Debug("file status checked", slog.Bool("processed", count > 0))
	return count > 0, nil
}

// ----------------------------------------------------------------------------
// Device messages methods
// ----------------------------------------------------------------------------

// SaveMessages - сохраняет сообщения устройств
func (r *Repository) SaveMessages(ctx context.Context, messages []models.DeviceMessage) error {
	const op = "postgres.SaveMessages"

	logger := r.logger.With(
		slog.String("op", op),
		slog.Int("batch_size", len(messages)),
	)

	if len(messages) == 0 {
		logger.Warn("no messages to save")
		return nil
	}

	logger.Info("saving messages to database")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Insert("device_messages").
		Columns(
			"number", "mqtt", "invid", "unit_guid", "message_id", "message_text",
			"context", "message_class", "level", "area", "address", "created_at",
		)

	for _, msg := range messages {
		query = query.Values(
			msg.Number,
			msg.Mqtt,
			msg.Invid,
			msg.UnitGUID,
			msg.MessageID,
			msg.MessageText,
			msg.Context,
			msg.MessageClass,
			msg.Level,
			msg.Area,
			msg.Address,
			time.Now(),
		)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	_, err = r.pool.Exec(ctx, sql, args...)
	if err != nil {
		logger.Error("failed to save messages", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	logger.Info("messages saved successfully", slog.Int("saved", len(messages)))
	return nil
}

// GetAllMessagesByUnitGUID - возвращает все сообщения устройства
func (r *Repository) GetAllMessagesByUnitGUID(ctx context.Context, unitGUID string) ([]models.DeviceMessage, error) {
	const op = "postgres.GetAllMessagesByUnitGUID"

	logger := r.logger.With(
		slog.String("op", op),
		slog.String("unit_guid", unitGUID),
	)

	logger.Info("getting all messages for device")

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Select(
			"number", "mqtt", "invid", "unit_guid", "message_id", "message_text",
			"context", "message_class", "level", "area", "address", "created_at",
		).
		From("device_messages").
		Where(sq.Eq{"unit_guid": unitGUID}).
		OrderBy("created_at DESC").
		ToSql()

	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		logger.Error("failed to query messages", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []models.DeviceMessage

	for rows.Next() {
		var msg models.DeviceMessage
		var createdAt time.Time

		err := rows.Scan(
			&msg.Number,
			&msg.Mqtt,
			&msg.Invid,
			&msg.UnitGUID,
			&msg.MessageID,
			&msg.MessageText,
			&msg.Context,
			&msg.MessageClass,
			&msg.Level,
			&msg.Area,
			&msg.Address,
			&createdAt,
		)
		if err != nil {
			logger.Error("failed to scan row", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}

		messages = append(messages, msg)
	}

	logger.Info("messages retrieved", slog.Int("count", len(messages)))
	return messages, nil
}

// GetMessagesByUnitGUIDWithPagination - возвращает сообщения с пагинацией
func (r *Repository) GetMessagesByUnitGUIDWithPagination(
	ctx context.Context,
	unitGUID string,
	page, limit int,
) ([]models.DeviceMessage, int, error) {
	const op = "postgres.GetMessagesByUnitGUIDWithPagination"

	logger := r.logger.With(
		slog.String("op", op),
		slog.String("unit_guid", unitGUID),
		slog.Int("page", page),
		slog.Int("limit", limit),
	)

	logger.Info("getting messages with pagination")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	countQuery, countArgs, err := psql.
		Select("COUNT(*)").
		From("device_messages").
		Where(sq.Eq{"unit_guid": unitGUID}).
		ToSql()

	if err != nil {
		logger.Error("failed to build count query", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: build count query: %w", op, err)
	}

	var total int
	err = r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		logger.Error("failed to get total count", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: count query: %w", op, err)
	}

	query, args, err := psql.
		Select(
			"number", "mqtt", "invid", "unit_guid", "message_id", "message_text",
			"context", "message_class", "level", "area", "address", "created_at",
		).
		From("device_messages").
		Where(sq.Eq{"unit_guid": unitGUID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()

	if err != nil {
		logger.Error("failed to build query", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: build query: %w", op, err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		logger.Error("failed to query messages", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []models.DeviceMessage

	for rows.Next() {
		var msg models.DeviceMessage
		var createdAt time.Time

		err := rows.Scan(
			&msg.Number,
			&msg.Mqtt,
			&msg.Invid,
			&msg.UnitGUID,
			&msg.MessageID,
			&msg.MessageText,
			&msg.Context,
			&msg.MessageClass,
			&msg.Level,
			&msg.Area,
			&msg.Address,
			&createdAt,
		)
		if err != nil {
			logger.Error("failed to scan row", slog.String("error", err.Error()))
			return nil, 0, fmt.Errorf("%s: scan: %w", op, err)
		}

		messages = append(messages, msg)
	}

	logger.Info("messages retrieved with pagination",
		slog.Int("count", len(messages)),
		slog.Int("total", total),
		slog.Int("pages", (total+limit-1)/limit),
	)

	return messages, total, nil
}
