-- piko.query(VisitCount, one)
SELECT count(*) AS total FROM visits WHERE path = {path:String};
