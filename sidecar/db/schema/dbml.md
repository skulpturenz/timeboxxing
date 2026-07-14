DBML:

Reflects the schema as implemented in `db/schema/*.sql`. Conventions: every `id` is an
`INTEGER PRIMARY KEY` (SQLite rowid alias, auto-increments, no `AUTOINCREMENT`); named `unique_*`
constraints enforce natural keys. Deleting a `foreground_processes` row cascades to everything linked
directly and indirectly. Referential actions (`delete: cascade`) are defined as standalone `Ref`
blocks at the end, since DBML does not support relationship settings on inline column refs.

```
// Event store
Table foreground_processes {
  id BIGINT [pk] // auto increment
  application_id BIGINT [ref: > applications.id]
  pid BIGINT [NOT NULL]
  created_at_utc TIMESTAMP [NOT NULL]

  indexes {
    created_at_utc [unique, name: 'unique_created_at_utc']
  }
}

TABLE foreground_process_metadata {
  id BIGINT [pk] // auto increment
  foreground_process_id BIGINT [NOT NULL]
  browser boolean [default: FALSE]
  idle boolean [default: FALSE]
  tab TEXT
  cdp_url TEXT // chrome dev tools protocol
  latitude REAL
  longitude REAL
  public_ip TEXT

  indexes {
    foreground_process_id [unique, name: 'unique_foreground_process_id']
  }
}

// Timeline
TABLE timeline {
  id BIGINT [pk] // auto increment
  initial_foreground_process_id BIGINT
  end_foreground_process_id BIGINT
}

TABLE timeline_semantic_documents {
  id BIGINT [pk] // auto increment
  timeline_id BIGINT
  type SMALLINT [ref: > semantic_document_types.id]
  content TEXT [NOT NULL]
  document_key TEXT [NOT NULL] // upsert key: "event:<timeline id>", "day:<date>", ...

  indexes {
    document_key [unique, name: 'unique_document_key']
  }
}

TABLE timeline_embeddings {
  id BIGINT [pk] // auto increment
  timeline_id BIGINT
  timeline_semantic_documents_id BIGINT
  embedding_model_id SMALLINT [ref: > models.id]
  dimension SMALLINT [NOT NULL]
  embedding BLOB [NOT NULL]

  indexes {
    (timeline_semantic_documents_id, embedding_model_id) [unique, name: 'unique_timeline_semantic_documents_id_embedding_model_id']
  }
}

// Entries
TABLE ledger {
  id BIGINT [pk] // auto increment
  created_at_utc TIMESTAMP [NOT NULL]
  updated_at_utc TIMESTAMP [NOT NULL]
}

TABLE ledger_items {
  id BIGINT [pk] // auto increment
  ledger_id BIGINT [ref: > ledger.id]
  billable BOOLEAN [default: FALSE]
  title TEXT [NOT NULL]
  notes TEXT
  started_at_utc TIMESTAMP
  ended_at_utc TIMESTAMP
  // CHECK (started_at_utc IS NULL OR ended_at_utc IS NULL OR ended_at_utc > started_at_utc)
}

TABLE ledger_item_timeline_entries {
  id BIGINT [pk] // auto increment
  ledger_items_id BIGINT
  timeline_id BIGINT
}

TABLE projects {
  id BIGINT [pk] // auto increment
  name TEXT [NOT NULL]

  indexes {
    `name COLLATE NOCASE` [unique, name: 'unique_name']
  }
}

TABLE project_costs {
  id BIGINT [pk] // auto increment
  ledger_items_id BIGINT // a ledger item may have many costs (assignable to multiple projects)
  projects_id BIGINT // item <-> project link
  costing_type_id SMALLINT [ref: > project_costing_types.id]
  rate INT
}

TABLE project_details {
  id BIGINT [pk] // auto increment
  projects_id BIGINT
  project_colors_id SMALLINT [ref: > project_colors.id]
  costing_type_id SMALLINT [ref: > project_costing_types.id]
  rate INT

  indexes {
    projects_id [unique, name: 'unique_projects_id']
  }
}

// App settings
TABLE application_settings {
  id SMALLINT [pk, NOT NULL] // singleton, CHECK (id = 1)
  model_provider_id SMALLINT [ref: > model_providers.id]
  model_provider_base_url TEXT
  embedding_model_id SMALLINT [ref: > models.id, NOT NULL]
  semantic_model_id SMALLINT [ref: > models.id, NOT NULL]
  release_channel SMALLINT [ref: > release_channels.id] // knowing the release channel helps with feature toggles if we ever do introduce them
}

// Reference data
Table applications {
  id BIGINT [pk] // auto increment
  name TEXT [NOT NULL]
  operating_system_id SMALLINT [ref: > operating_systems.id]
  path TEXT

  indexes {
    name [unique, name: 'unique_name']
  }
}

Table operating_systems {
  id SMALLINT [pk] // auto increment
  code TEXT [NOT NULL]
  label TEXT [NOT NULL]

  indexes {
    (id, code) [unique, name: 'unique_id_code']
  }
}

TABLE project_colors {
  id SMALLINT [pk] // auto increment
  color INTEGER [NOT NULL]
  description TEXT [NOT NULL]

  indexes {
    color [unique, name: 'unique_color']
  }
}

TABLE project_costing_types {
  id SMALLINT [pk] // auto increment
  label TEXT

  indexes {
    label [unique, name: 'unique_label']
  }
}

TABLE model_providers {
  id SMALLINT [pk, NOT NULL] // auto increment
  label TEXT [NOT NULL]

  indexes {
    label [unique, name: 'unique_label']
  }
}

TABLE models {
  id SMALLINT [pk] // auto increment
  semantic BOOLEAN [default: FALSE]
  embedding BOOLEAN [default: FALSE]
  openrouter_slug TEXT
  ollama_slug TEXT
  label TEXT [NOT NULL]
}

TABLE release_channels {
  id SMALLINT [pk] // auto increment
  label TEXT
}

TABLE semantic_document_types {
  id SMALLINT [pk] // auto increment
  code TEXT

  indexes {
    code [unique, name: 'unique_code']
  }
}

// Foreign keys with ON DELETE CASCADE (referential actions require standalone Ref definitions)
Ref: foreground_process_metadata.foreground_process_id - foreground_processes.id [delete: cascade]
Ref: timeline.initial_foreground_process_id > foreground_processes.id [delete: cascade]
Ref: timeline.end_foreground_process_id > foreground_processes.id [delete: cascade]
Ref: timeline_semantic_documents.timeline_id > timeline.id [delete: cascade]
Ref: timeline_embeddings.timeline_id > timeline.id [delete: cascade]
Ref: timeline_embeddings.timeline_semantic_documents_id > timeline_semantic_documents.id [delete: cascade]
Ref: ledger_item_timeline_entries.ledger_items_id > ledger_items.id [delete: cascade]
Ref: ledger_item_timeline_entries.timeline_id > timeline.id [delete: cascade]
Ref: project_costs.ledger_items_id > ledger_items.id [delete: cascade]
Ref: project_costs.projects_id > projects.id [delete: cascade]
Ref: project_details.projects_id - projects.id [delete: cascade]
```
