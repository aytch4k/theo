CREATE TABLE IF NOT EXISTS chains (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chain_type TEXT NOT NULL,
  repo_id TEXT NOT NULL,
  block_count INTEGER NOT NULL DEFAULT 0,
  last_updated TIMESTAMP NOT NULL,
  ipfs_cid TEXT,
  UNIQUE(chain_type, repo_id)
);

CREATE TABLE IF NOT EXISTS blocks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chain_type TEXT NOT NULL,
  repo_id TEXT NOT NULL,
  block_hash TEXT NOT NULL,
  block_index INTEGER NOT NULL,
  previous_hash TEXT NOT NULL,
  timestamp INTEGER NOT NULL,
  data TEXT NOT NULL,
  signature BLOB,
  metadata TEXT,
  context TEXT NOT NULL,
  block_id TEXT NOT NULL,
  UNIQUE(chain_type, repo_id, block_hash),
  FOREIGN KEY(chain_type, repo_id) REFERENCES chains(chain_type, repo_id)
);

INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid)
VALUES ('test-chain', 'test-repo', 1, datetime('now'), 'QmTestCID123456789');

INSERT INTO blocks (chain_type, repo_id, block_hash, block_index, previous_hash, timestamp, data, context, block_id)
VALUES (
  'test-chain',
  'test-repo',
  '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
  0,
  '',
  1710425600,
  '{"message":"Test data for SQLite storage","timestamp":1710425600,"index":0}',
  'https://devhub-git.org/contexts/block.jsonld',
  'chain://test-chain/test-repo/block0'
);
