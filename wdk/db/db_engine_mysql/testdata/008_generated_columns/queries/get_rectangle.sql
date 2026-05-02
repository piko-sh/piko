-- piko.query(name: GetRectangle, command: one)
SELECT id, width, height, area, perimeter FROM rectangles WHERE id = ?;
