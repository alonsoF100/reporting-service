package postgres

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upProcessedFiles, downProcessedFiles)
}

func upProcessedFiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE processed_files (
			id SERIAL PRIMARY KEY,
			file_name VARCHAR(255) NOT NULL UNIQUE,
			status VARCHAR(50) NOT NULL DEFAULT 'processing',
			error_message TEXT,
			processed_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_processed_files_status ON processed_files(status);
		CREATE INDEX idx_processed_files_file_name ON processed_files(file_name);
		CREATE INDEX idx_processed_files_processed_at ON processed_files(processed_at);
	`)
	return err
}

func downProcessedFiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE processed_files;`)
	return err
}
