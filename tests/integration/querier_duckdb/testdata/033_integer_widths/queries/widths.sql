-- piko.query(name: GetWidths, command: one)
SELECT id, tiny, small, regular, big, utiny, usmall, uregular, ubig, huge, uhuge
FROM widths
WHERE id = $1;
