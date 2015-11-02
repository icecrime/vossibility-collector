package main

import "testing"

var testRepository = Repository{
	RepositoryConfig: RepositoryConfig{
		User:  "icecrime",
		Repo:  "repository",
		Topic: "topic",
	},
	GivenName:    "testrepo",
	PeriodicSync: "daily",
}

type indexCall struct {
	Destination string
	Blob        *Blob
}

type testIndexer []indexCall

func (t *testIndexer) Index(destination string, blob *Blob) error {
	*t = append(*t, indexCall{destination, blob})
	return nil
}

func (t *testIndexer) Reset() {
	*t = []indexCall{}
}

func TestSimpleBlobStoreLive(t *testing.T) {
	s := simpleBlobStore{}
	indexer := testIndexer{}
	s.indexer = &indexer

	b := NewBlob("event", "id")
	if err := s.Store(StoreLiveEvent, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if len(indexer) != 1 {
		t.Fatalf("indexer was called %d times, expected once", len(indexer))
	}
}

func TestSimpleBlobStoreLiveWithSnapshot(t *testing.T) {
	s := simpleBlobStore{}
	indexer := testIndexer{}
	s.indexer = &indexer

	b := NewBlob("event", "id")
	b.Push(MetadataSnapshotID, "snapshot_id")
	b.Push(MetadataSnapshotField, "snapshot_field")
	if err := s.Store(StoreLiveEvent, &testRepository, b); err != nil {
		t.Fatalf("failed to store blob: %v", err)
	}
	if len(indexer) != 3 {
		t.Fatalf("indexer was called %d times, expected 3", len(indexer))
	}
}
