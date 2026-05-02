-- piko.query(name: ItemsByColour, command: many)
SELECT id, colour FROM items WHERE colour = $1::colour;
