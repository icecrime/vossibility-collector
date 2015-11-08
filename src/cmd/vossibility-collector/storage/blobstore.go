package storage

import (
	"fmt"

	"cmd/vossibility-collector/blob"
	"cmd/vossibility-collector/config"
	"cmd/vossibility-collector/transformation"

	log "github.com/Sirupsen/logrus"
)

// Storage is the target storage for an Index operation.
type Storage int

const (
	// StoreSnapshot corresponds to the non-expiring index (that is, the index
	// that always holds the latest version of all GitHub items). This index
	// only stores snapshotted data.
	StoreSnapshot Storage = iota

	// StoreCurrentState corresponds to the periocally rolling index (that is,
	// the index that gets archived and renewed at a regular interval). This
	// index only stores snapshotted data.
	StoreCurrentState

	// StoreLiveEvent corresponds to the rolling index of events.
	StoreLiveEvent
)

// BlobStore determines from the Storage and Repository how the Blob should be
// indexer to a backing blobIndexer. In the process, it might alter the Blob,
// for example in order to apply transformations.
type BlobStore interface {
	// Store saved the blob into the specified storage under the provided id
	// for a given repository.
	Store(Storage, *Repository, *blob.Blob) error
}

// transformingBlobStore applies transformations before forwarding the
// resulting blob to a simpleBlobStore.
type transformations struct {
	transformations transformation.Transformations
}

// NewTransformingBlobStore creates a new transformingBlobStore backed by a
// simpleBlobStore.
func NewTransformingBlobStore() BlobStore {
	return &transformingBlobStore{
		impl: NewSimpleBlobStore(),
	}
}

// transformingBlobStore implements BlobStore by applying transformations
// before forwarding the resulting blob to a backing BlobStore instance.
type transformingBlobStore struct {
	impl BlobStore
}

// Index stores the blob into the specified storage under the provided id for
// a given repository.
func (b *transformingBlobStore) Store(storage Storage, repo *Repository, blob *blob.Blob) error {
	if trans := b.getTransformation(storage, repo, blob.Type); trans != nil {
		ctx := transformation.Context{Repository: repo}
		t, err := trans.Apply(ctx, blob)
		if err != nil {
			return fmt.Errorf("applying transformation to event %q: %v", blob.Type, err)
		}
		blob = t
	}

	// Forward to the backing implementation.
	return b.impl.Store(storage, repo, blob)
}

func (b *transformingBlobStore) getTransformation(storage Storage, repo *Repository, event string) *transformation.Transformation {
	// Live and snapshot data have overlapping types: we can received a
	// "pull_request" live event for a new pull request being opened, as well
	// as a "pull_request" snapshot during a sync operation.
	//
	// This is problematic, because different transformations have to be
	// applies, which is why the storage type contributes to the transformation
	// election.
	if storage == StoreLiveEvent {
		return repo.EventSet[event]
	}

	// This is not a live event: we have hardcoded transformations for the
	// issues and pull requests data types.
	switch event {
	case config.GitHubTypeIssue:
		return repo.EventSet[config.SnapshotIssueType]
	case config.GitHubTypePullRequest:
		return repo.EventSet[config.SnapshotPullRequestType]
	default:
		// No transformation for that event type.
		log.Warnf("no transformation found for event type %q", event)
		return nil
	}
}

// NewSimpleBlobStore creates a new simpleBlobStore.
func NewSimpleBlobStore() BlobStore {
	return &simpleBlobStore{
		indexer: elasticSearchIndexer{},
	}
}

// simpleBlobStore provides basic facilities for writing into Elastic Search.
type simpleBlobStore struct {
	indexer blobIndexer
}

// Index stores the blob into the specified storage under the provided id for
// a given repository.
func (b *simpleBlobStore) Store(storage Storage, repo *Repository, blob *blob.Blob) error {
	switch storage {
	// Live is an index containing the webhook events. In this particular case,
	// we use the delivery id as the document index.
	//
	// When storing a live event, we always update the next two indices.
	case StoreLiveEvent:
		liveIndex := repo.LiveIndexForTimestamp(blob.Timestamp)
		log.Debugf("store live event to %s/%s/%s", liveIndex, blob.Type, blob.ID)
		if err := b.indexer.Index(liveIndex, blob); err != nil {
			return fmt.Errorf("store live event %s data: %v", blob.ID, err)
		}
		// Before falling through, replace the blob with the snapshot data from
		// the event, if any.
		if blob = blob.Snapshot(); blob == nil {
			return nil
		}
		fallthrough
	// Current state is an index containing the last version of items at a
	// given moment in time, and is updated at a frequency configured by the
	// user.
	//
	// When storing a current state, we always update the next index.
	case StoreCurrentState:
		stateIndex := repo.StateIndexForTimestamp(blob.Timestamp)
		log.Debugf("store current state to %s/%s/%s", stateIndex, blob.Type, blob.ID)
		if err := b.indexer.Index(stateIndex, blob); err != nil {
			return fmt.Errorf("store current state %s data: %v", blob.ID, err)
		}
		fallthrough
	// Snapshot is an index containing the last version of all items, opened or
	// closed.
	case StoreSnapshot:
		log.Debugf("store snapshot to %s/%s/%s", repo.SnapshotIndex(), blob.Type, blob.ID)
		if err := b.indexer.Index(repo.SnapshotIndex(), blob); err != nil {
			return fmt.Errorf("store snapshot %s data: %v", blob.ID, err)
		}
	}
	return nil
}
