package postgres

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDeviceMessages, downDeviceMessages)
}

func upDeviceMessages(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE device_messages (
			id SERIAL PRIMARY KEY,
			number INTEGER,
			mqtt VARCHAR(100),
			invid VARCHAR(50),
			unit_guid UUID NOT NULL,
			message_id VARCHAR(255) NOT NULL,
			message_text TEXT,
			context VARCHAR(100),
			message_class VARCHAR(50),
			level INTEGER,
			area VARCHAR(50),
			address TEXT,
			source_file VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_device_messages_unit_guid ON device_messages(unit_guid);
		CREATE INDEX idx_device_messages_message_class ON device_messages(message_class);
		CREATE INDEX idx_device_messages_created_at ON device_messages(created_at);
		CREATE INDEX idx_device_messages_invid ON device_messages(invid);
		CREATE INDEX idx_device_messages_source_file ON device_messages(source_file);
	`)
	return err
}

func downDeviceMessages(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE device_messages;`)
	return err
}