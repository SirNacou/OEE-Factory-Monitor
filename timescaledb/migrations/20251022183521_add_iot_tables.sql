-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS production_events (
    time timestamptz NOT NULL,
    machine_id integer NOT NULL,
    parts_produced integer NOT NULL,
    parts_scrapped integer NOT NULL
  )
WITH
  (tsdb.hypertable, tsdb.partition_column = 'time');

CREATE TABLE IF NOT EXISTS status_events (
    time timestamptz NOT NULL,
    machine_id integer NOT NULL,
    status text NOT NULL
  )
WITH
  (tsdb.hypertable, tsdb.partition_column = 'time');

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS production_events;

DROP TABLE IF EXISTS status_events;

-- +goose StatementEnd