package main

import (
	"time"

	"github.com/mattbaird/elastigo/core"
)

// blobIndexer does the lower level blob indexing.
type blobIndexer interface {
	Index(destination string, blob *Blob) error
}

// elasticSearchIndexer implements blobIndexer by storing to an ElasticSearch
// backend.
type elasticSearchIndexer struct{}

// Index stores a blob into the specific index.
func (elasticSearchIndexer) Index(index string, blob *Blob) error {
	// Apparently Elastic Search don't like timezone specifiers other than Z.
	timestamp := blob.Timestamp.UTC().Format(time.RFC3339)
	//log.Warnf("Index [%s] add [%s] [%s] [%#v]\n", index, blob.ID, timestamp, *blob.Data)
	_, err := core.IndexWithParameters(
		index, blob.Type, blob.ID,
		"" /* parentId */, 0 /* version */, "" /* op_type */, "", /* routing */
		timestamp,
		0 /* ttl */, "" /* percolate */, "" /* timeout */, false /* refresh */, map[string]interface{}{}, /* args */
		blob.Data)
	return err
}
