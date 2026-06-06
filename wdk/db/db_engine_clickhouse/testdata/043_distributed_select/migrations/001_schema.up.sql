CREATE TABLE distributed_t (id UInt64, val String) ENGINE = Distributed('cluster1', 'db', 'local_t', id);
