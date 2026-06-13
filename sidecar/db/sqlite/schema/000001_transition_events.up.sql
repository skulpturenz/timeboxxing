CREATE TABLE applications (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE transition_event_reasons (
  id INTEGER PRIMARY KEY NOT NULL,
  reason TEXT NOT NULL UNIQUE
);

CREATE TABLE transition_events (
  id INTEGER PRIMARY KEY,
  application_id INTEGER REFERENCES applications(id),
  transition_reason_id INTEGER NOT NULL REFERENCES transition_event_reasons(id),
  started_at TIMESTAMP NOT NULL,
  ended_at TIMESTAMP NOT NULL
);

CREATE TABLE transition_event_metadata (
  id INTEGER PRIMARY KEY,
  transition_event_id INTEGER NOT NULL REFERENCES transition_events(id),
  browser BOOLEAN NOT NULL DEFAULT false,
  tab TEXT,
  idle BOOLEAN NOT NULL DEFAULT false,
  cdp_url TEXT
);
