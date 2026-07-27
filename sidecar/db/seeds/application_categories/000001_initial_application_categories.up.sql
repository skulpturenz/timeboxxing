INSERT INTO application_categories (id, category_id, code, label)
VALUES
  (1, 0, 'unknown', 'Unknown'),
  (2, 1, 'development', 'Development'),
  (3, 2, 'productivity', 'Productivity'),
  (4, 3, 'communication', 'Communication'),
  (5, 4, 'web-browsing', 'Web Browsing'),
  (6, 5, 'media', 'Media & Entertainment'),
  (7, 6, 'graphics-design', 'Graphics & Design'),
  (8, 7, 'games', 'Games'),
  (9, 8, 'utilities', 'Utilities'),
  (10, 9, 'business-finance', 'Business & Finance'),
  (11, 10, 'education', 'Education'),
  (12, 11, 'social', 'Social Networking'),
  (13, 12, 'system', 'System'),
  (14, 13, 'other', 'Other')
ON CONFLICT(category_id) DO UPDATE SET
  code = excluded.code,
  label = excluded.label;
