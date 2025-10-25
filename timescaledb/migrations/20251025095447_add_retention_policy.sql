-- +goose Up
-- +goose StatementBegin
SELECT
  add_retention_policy ('production_events', INTERVAL '30 days');

SELECT
  add_retention_policy ('status_events', INTERVAL '30 days');

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
SELECT
  remove_retention_policy ('production_events');

SELECT
  remove_retention_policy ('status_events');

-- +goose StatementEnd