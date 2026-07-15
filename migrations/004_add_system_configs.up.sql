CREATE TABLE IF NOT EXISTS security.system_configs (
    key   TEXT PRIMARY KEY,
    value JSONB NOT NULL
);

INSERT INTO security.system_configs (key, value)
VALUES ('default_categories', '[
  { "name": "Salario",      "type": "income",  "icon_key": "salary",    "color": "#16A34A" },
  { "name": "Ventas",       "type": "income",  "icon_key": "sales",     "color": "#0EA5E9" },
  { "name": "Alimentación", "type": "expense", "icon_key": "food",      "color": "#F97316" },
  { "name": "Servicios",    "type": "expense", "icon_key": "services",  "color": "#3B82F6" },
  { "name": "Salud",        "type": "expense", "icon_key": "health",    "color": "#EF4444" },
  { "name": "Transporte",   "type": "expense", "icon_key": "transport", "color": "#EAB308" },
  { "name": "Hogar",        "type": "expense", "icon_key": "home",      "color": "#A855F7" }
]')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
