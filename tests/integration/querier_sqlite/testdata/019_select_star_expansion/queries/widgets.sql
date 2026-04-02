-- piko.name: GetWidget
-- piko.command: one
SELECT * FROM widgets WHERE id = ?;

-- piko.name: GetWidgetQualified
-- piko.command: one
SELECT w.* FROM widgets w WHERE w.id = ?;
