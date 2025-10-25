-- +goose Up
-- +goose StatementBegin
CREATE TABLE
  machines (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ideal_cycle_time_sec FLOAT NOT NULL,
    -- NEW: Add default plan values
    default_target_count INT NOT NULL DEFAULT 1000,
    default_planned_downtime_min INT NOT NULL DEFAULT 45
  );

-- Re-insert your machines with the new default values
INSERT INTO
  machines (
    id,
    name,
    ideal_cycle_time_sec,
    default_target_count,
    default_planned_downtime_min
  )
VALUES
  (1, 'CNC Machine 1', 3.0, 9000, 45),
  (2, 'Stamping Press 2', 3.0, 9000, 60),
  (3, 'Assembly Line 3', 3.0, 7500, 30);

CREATE TABLE
  shifts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL
    -- We can also add default planned break times here
    -- default_planned_break_min INT DEFAULT 30
  );

-- Insert your shift patterns
INSERT INTO
  shifts (name, start_time, end_time)
VALUES
  ('Day Shift', '07:00:00', '15:00:00'),
  ('Night Shift', '15:00:00', '23:00:00'),
  ('Graveyard Shift', '23:00:00', '07:00:00');

CREATE TABLE
  production_plan (
    plan_date DATE NOT NULL,
    machine_id INT REFERENCES machines (id),
    shift_id INT REFERENCES shifts (id),
    -- This is your "target product count"
    target_count INT NOT NULL,
    -- This is crucial for real OEE!
    -- How much downtime is PLANNED? (e.g., setup, cleaning, lunch)
    planned_downtime_min INT NOT NULL DEFAULT 60,
    UNIQUE (machine_id, shift_id, plan_date)
  )
WITH
  (
    tsdb.hypertable,
    tsdb.partition_column = 'plan_date',
    tsdb.chunk_interval = '1 month'
  );

SELECT
  add_dimension ('production_plan', by_hash ('machine_id', 4));

SELECT
  add_dimension ('production_plan', by_hash ('shift_id', 3));

CREATE UNIQUE INDEX ON production_plan (machine_id, shift_id, plan_date);

-- Example: Schedule Machine 1 for today's Day Shift
-- Machine 1, Day Shift, Today, Target 8000, 45 min planned downtime
INSERT INTO
  production_plan (
    machine_id,
    shift_id,
    plan_date,
    target_count,
    planned_downtime_min
  )
VALUES
  (1, 1, CURRENT_DATE, 8000, 45);

SELECT
  add_retention_policy ('production_plan', INTERVAL '90 days');

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS production_plan;

DROP TABLE IF EXISTS machines;

DROP TABLE IF EXISTS shifts;

-- +goose StatementEnd