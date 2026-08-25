-- +goose Up
CREATE TABLE nodes (
  id                TEXT PRIMARY KEY,
  title             TEXT NOT NULL,
  content_md        TEXT NOT NULL DEFAULT '',
  status            TEXT NOT NULL DEFAULT 'not_started'
                    CHECK (status IN ('not_started','learning','partial','mastered','forgotten')),
  stage             TEXT,
  sort_order        REAL NOT NULL DEFAULT 0,
  parent_id         TEXT REFERENCES nodes(id) ON DELETE CASCADE,
  pos_x             REAL,
  pos_y             REAL,
  status_changed_at INTEGER,
  source_note       TEXT,
  created_at        INTEGER NOT NULL,
  updated_at        INTEGER NOT NULL
);
CREATE INDEX idx_nodes_parent ON nodes(parent_id);
CREATE INDEX idx_nodes_status ON nodes(status);

CREATE TABLE edges (
  id         TEXT PRIMARY KEY,
  source_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  target_id  TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  relation   TEXT NOT NULL DEFAULT 'related'
             CHECK (relation IN ('prerequisite','related')),
  label      TEXT,
  created_at INTEGER NOT NULL,
  UNIQUE (source_id, target_id, relation)
);
CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);

CREATE TABLE resources (
  id         TEXT PRIMARY KEY,
  node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  kind       TEXT NOT NULL DEFAULT 'link',
  title      TEXT NOT NULL,
  url        TEXT,
  path       TEXT,
  note       TEXT,
  created_at INTEGER NOT NULL
);
CREATE INDEX idx_resources_node ON resources(node_id);

CREATE TABLE exercises (
  id            TEXT PRIMARY KEY,
  node_id       TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  type          TEXT NOT NULL CHECK (type IN ('single_choice','multiple_choice','true_false','fill_blank','short_answer')),
  question_md   TEXT NOT NULL,
  options_json  TEXT,
  answer_md     TEXT NOT NULL,
  analysis_md   TEXT,
  difficulty    INTEGER DEFAULT 3,
  answer_draft  TEXT,
  result        TEXT CHECK (result IN ('right','partial','wrong')),
  score         INTEGER,
  feedback_md   TEXT,
  answered_at   INTEGER,
  created_at    INTEGER NOT NULL
);
CREATE INDEX idx_exercises_node ON exercises(node_id);

CREATE TABLE annotations (
  id         TEXT PRIMARY KEY,
  node_id    TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  content_md TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_annotations_node ON annotations(node_id);

CREATE TABLE settings (
  key        TEXT PRIMARY KEY,
  value_json TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS annotations;
DROP TABLE IF EXISTS exercises;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS edges;
DROP TABLE IF EXISTS nodes;
