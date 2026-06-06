-- piko.query(GetHash, one)
SELECT id, sha1, sha256 FROM hashes WHERE id = {id:UInt64};
