-- piko.name: GetConfig
-- piko.command: one
SELECT id, name, CAST(settings AS VARCHAR) AS settings FROM configs WHERE id = $1;

-- piko.name: ListConfigs
-- piko.command: many
SELECT id, name, CAST(settings AS VARCHAR) AS settings FROM configs ORDER BY id;
