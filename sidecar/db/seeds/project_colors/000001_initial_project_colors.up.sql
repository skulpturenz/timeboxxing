INSERT INTO project_colors (id, color, description)
VALUES
  (1, 0xFF00FFEE, 'Teal'),
  (2, 0xFF4F7CFF, 'Blue'),
  (3, 0xFF33B679, 'Green'),
  (4, 0xFFFFB020, 'Amber'),
  (5, 0xFFE25563, 'Red'),
  (6, 0xFF9B6DFF, 'Purple'),
  (7, 0xFFFF7A45, 'Orange'),
  (8, 0xFF2FA7B8, 'Cyan'),
  (9, 0xFF6E7F80, 'Slate')
ON CONFLICT(id) DO UPDATE SET
  color = excluded.color,
  description = excluded.description;
