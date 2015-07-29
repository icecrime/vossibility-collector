package main

import (
	"fmt"
	"strconv"

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

func (b *blobStore) Index(storage Storage, repo *Repository, blob *Blob, delivery string) error {
	// Apply the transformation.
	trans := b.Config.Transformations[blob.Type()]
	d, err := trans.ApplyBlob(blob)
	if err != nil {
		return fmt.Errorf("applying transformation to event %q: %v", blob.Type(), err)
	}

	switch storage {
	// Live is an index containing the webhook events. In this particular case,
	// we use the delivery id as the document index.
	//
	// When storing a live event, we always update the next two indices.
	case StoreLiveEvent:
		log.Debugf("store live event to %s/%s/%s", repo.LiveIndex(), blob.Type(), delivery)
		if _, err := core.Index(repo.LiveIndex(), d.Type(), delivery, nil, d.Data); err != nil {
			return fmt.Errorf("store live event %s data: %v", delivery, err)
		}
		fallthrough

	// Current state is an index containing the last version of items at a
	// given moment in time, and is updated at a frequency configured by the
	// user.
	//
	// When storing a current state, we always update the next index.
	case StoreCurrentState:
		if _, ok := GithubSnapshotedEvents[blob.Type()]; ok {
			log.Debugf("store current state to %s/%s/%d", repo.CurrentStateIndex(), blob.Type(), blob.Number())
			if _, err := core.Index(repo.CurrentStateIndex(), blob.Type(), strconv.Itoa(blob.Number()), nil, blob.Data); err != nil {
				return fmt.Errorf("store live event %s data: %v", delivery, err)
			}
		}
		fallthrough

	// Snapshot is an index containing the last version of all items, opened or
	// closed.
	case StoreSnapshot:
		if _, ok := GithubSnapshotedEvents[blob.Type()]; ok {
			log.Debugf("store snapshot to %s/%s/%d", repo.SnapshotIndex(), blob.Type(), blob.Number())
			if _, err := core.Index(repo.SnapshotIndex(), blob.Type(), strconv.Itoa(blob.Number()), nil, d.Data); err != nil {
				return fmt.Errorf("store live event %s data: %v", delivery, err)
			}
		}
	}

	return nil
}

func (b *blobStore) IndexSingle(blob *Blob, index string) error {
	// Apply the transformation.
	trans := b.Config.Transformations[blob.Type()]
	d, err := trans.ApplyBlob(blob)
	if err != nil {
		return fmt.Errorf("applying transformation to event %q: %v", blob.Type(), err)
	}

	// Send to Elastic Search.
	if _, err := core.Index(index, d.Type(), d.Id(), nil, d.Data); err != nil {
		return fmt.Errorf("store pull request %s data: %v", d.Id(), err)
	}
	return nil
}

func (b *blobStore) PrepareForStorage(repo *Repository, o *Blob) error {
	if o.Type() == EvtPullRequest && !o.HasAttribute(LabelsAttribute) {
		log.Debugf("fetching labels for %s #%d", repo.PrettyName(), o.Number())
		l, _, err := b.Client.Issues.ListLabelsByIssue(repo.User, repo.Repo, o.Number(), &github.ListOptions{})
		if err != nil {
			return fmt.Errorf("retrieve labels for issue %s: %v", o.Id(), err)
		}
		o.Push(LabelsAttribute, l)
	}
	return nil
}
