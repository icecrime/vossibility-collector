package storage

import (
	"testing"

	"cmd/vossibility-collector/blob"
	"cmd/vossibility-collector/config"
)

var testRepository = Repository{
	RepositoryConfig: config.RepositoryConfig{
		User:  "icecrime",
		Repo:  "repository",
		Topic: "topic",
	},
	GivenName:    "testrepo",
	PeriodicSync: "daily",
}

type indexCall struct {
	Destination string
	Blob        *blob.Blob
}

type testIndexer []indexCall

func (t *testIndexer) Index(destination string, blob *blob.Blob) error {
	*t = append(*t, indexCall{destination, blob})
	return nil
}

func (t *testIndexer) Len() int {
	return len(*t)
}

func (t *testIndexer) Reset() {
	*t = []indexCall{}
}

func simpleBlobStoreSetup() (BlobStore, *testIndexer) {
	s := simpleBlobStore{}
	indexer := testIndexer{}
	s.indexer = &indexer
	return &s, &indexer
}

func TestSimpleBlobStoreLiveWithoutSnapshot(t *testing.T) {
	s, indexer := simpleBlobStoreSetup()
	b := blob.NewBlob("event", "id")
	if err := s.Store(StoreLiveEvent, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if indexer.Len() != 1 {
		t.Fatalf("indexer was called %d times, expected once", indexer.Len())
	}
}

func TestSimpleBlobStoreCascading(t *testing.T) {
	s, indexer := simpleBlobStoreSetup()

	// Create a snapshottable blob.
	b := blob.NewBlob("event", "id")
	b.Push(config.MetadataSnapshotID, "snapshot_id")
	b.Push(config.MetadataSnapshotField, "snapshot_field")

	// Verify that storing to StoreSnapshot doesn't cascade to any other store.
	if err := s.Store(StoreSnapshot, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if indexer.Len() != 1 {
		t.Fatalf("indexer was called %d times, expected once", indexer.Len())
	}
	destSnapshot := (*indexer)[0].Destination

	// Verify that storing to StoreCurrentState cascades to StoreSnapshot using
	// the same destination than a direct call.
	indexer.Reset()
	if err := s.Store(StoreCurrentState, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if indexer.Len() != 2 {
		t.Fatalf("indexer was called %d times, expected once", indexer.Len())
	}
	destCurrent := (*indexer)[0].Destination
	if cascading := (*indexer)[1].Destination; cascading != destSnapshot {
		t.Fatalf("cascading current snapshot to %q, expected %q", cascading, destSnapshot)
	}

	// Verify that storing to StoreLiveEvent cascades to StoreCurrentState using
	// the same destination than a direct call, and to StoreSnapshot using the
	// same destination than a direct call.
	indexer.Reset()
	if err := s.Store(StoreLiveEvent, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if indexer.Len() != 3 {
		t.Fatalf("indexer was called %d times, expected once", indexer.Len())
	}
	if cascading := (*indexer)[1].Destination; cascading != destCurrent {
		t.Fatalf("cascading live event to %q, expected %q", cascading, destCurrent)
	}
	if cascading := (*indexer)[2].Destination; cascading != destSnapshot {
		t.Fatalf("cascading live event to %q, expected %q", cascading, destSnapshot)
	}
}
