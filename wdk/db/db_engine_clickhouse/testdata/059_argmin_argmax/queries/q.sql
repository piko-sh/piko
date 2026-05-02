-- piko.query(ExtremeLabels, one)
SELECT argMin(label, score) AS lowest, argMax(label, score) AS highest FROM t;
