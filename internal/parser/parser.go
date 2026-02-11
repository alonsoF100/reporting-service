package parser

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/alonsoF100/reporting-service/internal/models"
)

func ParseTSV(filePath string) (*models.ParseResult, error) {
	const op = "parser.ParseTSV"

	logger := slog.With(
		slog.String("op", op),
		slog.String("file", filePath),
	)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("failed to open file",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = 11 // всегда 11 колонок!

	records, err := reader.ReadAll()
	if err != nil {
		logger.Error("failed read CSV",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("%s: failed read CSV: %w", op, err)
	}

	logger.Info("File loaded",
		slog.Int("total_rows", len(records)),
	)

	if len(records) < 3 {
		logger.Error("file too short",
			slog.Int("rows", len(records)),
		)
		return nil, fmt.Errorf("%s: file too short, need at least 3 rows", op)
	}

	result := &models.ParseResult{
		FileName: filePath,
		Messages: []models.DeviceMessage{},
	}

	// Пропускаем первые 2 строки (описание и заголовки)
	for i := 2; i < len(records); i++ {
		record := records[i]

		if len(record) == 0 || (len(record) == 1 && record[0] == "") {
			continue
		}

		msg := models.DeviceMessage{
			Number:       parseInt(record[0]),
			Mqtt:         strings.TrimSpace(record[1]),
			Invid:        strings.TrimSpace(record[2]),
			UnitGUID:     strings.TrimSpace(record[3]),
			MessageID:    strings.TrimSpace(record[4]),
			MessageText:  strings.TrimSpace(record[5]),
			Context:      strings.TrimSpace(record[6]),
			MessageClass: strings.TrimSpace(record[7]),
			Level:        parseInt(record[8]),
			Area:         strings.TrimSpace(record[9]),
			Address:      strings.TrimSpace(record[10]),
		}

		result.Messages = append(result.Messages, msg)
	}

	logger.Info("parsing completed",
		slog.Int("parsed_messages", len(result.Messages)),
	)

	return result, nil
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
