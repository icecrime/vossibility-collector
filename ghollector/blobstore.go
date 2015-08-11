package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/mattbaird/elastigo/core"
)

const (
	LabelsAttribute = "labels"
)

type Storage int

const (
	StoreSnapshot Storage = iota
	StoreCurrentState
	StoreLiveEvent
)

type blobStore struct {
	Client *github.Client
	Config *Config
}

func (b *blobStore) Index(storage Storage, repo *Repository, blob *Blob, id string) error {
	// Apply the transformation.
	if trans, ok := b.Config.Transformations[blob.Type()]; ok {
		b, err := trans.ApplyBlob(blob)
		if err != nil {
			return fmt.Errorf("applying transformation to event %q: %v", blob.Type(), err)
		}
		blob = b
	}

	switch storage {
	// Live is an index containing the webhook events. In this particular case,
	// we use the delivery id as the document index.
	//
	// When storing a live event, we always update the next two indices.
	case StoreLiveEvent:
		log.Debugf("store live event to %s/%s/%s", repo.LiveIndex(), blob.Type(), id)
		if _, err := core.Index(repo.LiveIndex(), blob.Type(), id, nil, blob.Data); err != nil {
			return fmt.Errorf("store live event %s data: %v", id, err)
		}
		id, blob = blob.Snapshot()
		fallthrough
	// Current state is an index containing the last version of items at a
	// given moment in time, and is updated at a frequency configured by the
	// user.
	//
	// When storing a current state, we always update the next index.
	case StoreCurrentState:
		if _, ok := GithubSnapshotedEvents[blob.Type()]; ok {
			log.Debugf("store current state to %s/%s/%s", repo.CurrentStateIndex(), blob.Type(), id)
			if _, err := core.Index(repo.CurrentStateIndex(), blob.Type(), id, nil, blob.Data); err != nil {
				return fmt.Errorf("store current state %s data: %v", id, err)
			}
		}
		fallthrough
	// Snapshot is an index containing the last version of all items, opened or
	// closed.
	case StoreSnapshot:
		if _, ok := GithubSnapshotedEvents[blob.Type()]; ok {
			log.Debugf("store snapshot to %s/%s/%s", repo.SnapshotIndex(), blob.Type(), id)
			log.Debugf("%#v\n", blob.Data)
			if _, err := core.Index(repo.SnapshotIndex(), blob.Type(), id, nil, blob.Data); err != nil {
				return fmt.Errorf("store snapshot %s data: %v", id, err)
			}
		}
	}
	return nil
}

func (b *blobStore) PrepareForStorage(repo *Repository, o *Blob) error {
	if o.Type() == EvtPullRequest && !o.HasAttribute(LabelsAttribute) {
		number := o.Data.Get("number").MustInt()
		log.Debugf("fetching labels for %s #%d", repo.PrettyName(), number)
		l, _, err := b.Client.Issues.ListLabelsByIssue(repo.User, repo.Repo, number, &github.ListOptions{})
		if err != nil {
			return fmt.Errorf("retrieve labels for issue %s: %v", number, err)
		}
		o.Push(LabelsAttribute, l)
	}
	return nil
}
