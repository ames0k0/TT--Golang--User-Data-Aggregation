-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_subscriptions (
	id		UUID		PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id		UUID		NOT NULL,
	service_name	TEXT		NOT NULL,
	price		INT		NOT NULL,
	start_date	VARCHAR (7)	NOT NULL,
	end_date	VARCHAR (7)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_subscriptions;
-- +goose StatementEnd
