-- piko.query(name: UnnestNamesAndPrices, command: many)
SELECT * FROM UNNEST($1::text[], $2::numeric[]) AS t(name text, price numeric);
