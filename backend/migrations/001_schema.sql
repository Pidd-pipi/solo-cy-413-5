-- PostgreSQL schema is managed by GORM AutoMigrate; this file documents the production tables.
CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, nickname TEXT NOT NULL, avatar TEXT, birth_date DATE, gender TEXT, role TEXT NOT NULL DEFAULT 'user', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS moods (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id), mood_level INT NOT NULL CHECK(mood_level BETWEEN 1 AND 10), mood_tags TEXT NOT NULL, note TEXT, record_date TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS assessments (id BIGSERIAL PRIMARY KEY, title TEXT NOT NULL, description TEXT, category TEXT NOT NULL, questions TEXT NOT NULL, scoring_rule TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS journals (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id), title TEXT NOT NULL, content TEXT NOT NULL, mood_level INT, weather TEXT, is_private BOOL NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());

-- ---- 七日身心调整计划模块（GORM AutoMigrate 同步；以下为生产表结构文档）----
CREATE TABLE IF NOT EXISTS adjustment_plans (id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id), status TEXT NOT NULL DEFAULT 'active', start_date TIMESTAMPTZ NOT NULL, end_date TIMESTAMPTZ NOT NULL, finished_at TIMESTAMPTZ, report_json TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE TABLE IF NOT EXISTS plan_versions (id BIGSERIAL PRIMARY KEY, plan_id BIGINT NOT NULL REFERENCES adjustment_plans(id) ON DELETE CASCADE, version INT NOT NULL, status TEXT NOT NULL DEFAULT 'current', trigger TEXT NOT NULL DEFAULT 'init', input_hash TEXT NOT NULL, snapshot_json TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), replaced_at TIMESTAMPTZ);
CREATE UNIQUE INDEX IF NOT EXISTS uk_plan_version ON plan_versions(plan_id, version);
CREATE TABLE IF NOT EXISTS plan_days (id BIGSERIAL PRIMARY KEY, plan_id BIGINT NOT NULL REFERENCES adjustment_plans(id) ON DELETE CASCADE, version_id BIGINT NOT NULL REFERENCES plan_versions(id), day_index INT NOT NULL, day_date TIMESTAMPTZ NOT NULL, theme TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', note TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE UNIQUE INDEX IF NOT EXISTS uk_plan_day_index ON plan_days(plan_id, day_index);
CREATE TABLE IF NOT EXISTS plan_tasks (id BIGSERIAL PRIMARY KEY, day_id BIGINT NOT NULL REFERENCES plan_days(id) ON DELETE CASCADE, plan_id BIGINT NOT NULL REFERENCES adjustment_plans(id) ON DELETE CASCADE, slot TEXT NOT NULL, source TEXT NOT NULL DEFAULT 'system', title TEXT NOT NULL, content TEXT, status TEXT NOT NULL DEFAULT 'pending', user_content TEXT, version INT NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE UNIQUE INDEX IF NOT EXISTS uk_day_slot ON plan_tasks(day_id, slot);
CREATE INDEX IF NOT EXISTS idx_plan_tasks_plan ON plan_tasks(plan_id);
-- 每个账号最多一个进行中（active/paused）计划：并发与重复提交的最终防线
CREATE UNIQUE INDEX IF NOT EXISTS uk_plan_one_active ON adjustment_plans(user_id) WHERE status IN ('active','paused');
