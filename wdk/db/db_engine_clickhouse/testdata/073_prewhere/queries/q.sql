-- piko.query(RegionFiltered, many)
SELECT id, val FROM t PREWHERE region = {region:String} WHERE val > 0;
