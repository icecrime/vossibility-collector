package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

const (
	// MetadataType is the key for the type metadata attribute used when
	// storing a Blob instance into the Elastic Search backend.
	MetadataType = "_type"

	// MetadataSnapshotID is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the Id to be used when storing the snapshoted content of a
	// blob.
	MetadataSnapshotID = "_snapshot_id"

	// MetadataSnapshotField is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the nested object to be used when storing the snapshoted
	// content of a blob.
	MetadataSnapshotField = "_snapshot_field"
)

// Blob is a opaque type representing an arbitrary payload from GitHub.
type Blob struct {
	// Data is the payload content.
	Data *simplejson.Json

	// ID sets the blob primary key in the Elastic Search store.
	ID string

	// SnapshotID identifies the field of the snapshotted data that should be
	// used as the snapshot Id.
	SnapshotID string

	// SnapshotField identifies the field of the blob data that should be
	// extracted and used as a snapshot.
	SnapshotField string

	// Timestamp is the creation time of the blob. It defaults to the object
	// creation time but can be overriden, most notably in the case of life
	// events when the queue timestamp will be used instead.
	Timestamp time.Time

	// Type is the blob type in the Elastic Search store.
	Type string
}

// NewBlob returns an empty Blob for that particular event type and id.
func NewBlob(event, id string) *Blob {
	return NewBlobFromJSON(event, id, simplejson.New())
}

func NewBlobFromJSON(event, id string, json *simplejson.Json) *Blob {
	return &Blob{
		Data:      json,
		ID:        id,
		Timestamp: time.Now(),
		Type:      event,
	}
}

func NewBlobFromPayload(event, id string, payload []byte) (*Blob, error) {
	d, err := simplejson.NewJson(payload)
	if err != nil {
		return nil, err
	}
	return NewBlobFromJSON(event, id, d), nil
}

func (b *Blob) Encode() ([]byte, error) {
	return b.Data.Encode()
}

func (b *Blob) HasAttribute(attr string) bool {
	_, ok := b.Data.CheckGet(attr)
	return ok
}

func (b *Blob) Push(key string, value interface{}) error {
	if strings.HasPrefix(key, "_") {
		return b.pushSpecialAttribute(key, value)
	}
	path := strings.Split(key, ".")
	b.Data.SetPath(path, value)
	return nil
}

// Snapshot returns the ID and Data for the snapshot for a Blob that models a
// live event.
func (b *Blob) Snapshot() *Blob {
	if b.SnapshotID == "" || b.SnapshotField == "" {
		return nil
	}

	// The snapshot data is simply a sub-attribute of the blob data. Its id is
	// a sub-attribute of the result, as identified by SnapshotID.
	snapshot := b.Data.Get(b.SnapshotField)
	snapshotID := fmt.Sprintf("%v", snapshot.GetPath(strings.Split(b.SnapshotID, ".")...).Interface())
	return &Blob{
		Data:      snapshot,
		ID:        snapshotID,
		Timestamp: b.Timestamp,
		Type:      b.Type,
	}
}

func (b *Blob) pushSpecialAttribute(key string, value interface{}) error {
	metaFields := map[string]*string{
		MetadataType:          &b.Type,
		MetadataSnapshotID:    &b.SnapshotID,
		MetadataSnapshotField: &b.SnapshotField,
	}
	if target, ok := metaFields[key]; !ok {
		return fmt.Errorf("invalid metadata field %q", key)
	} else if strValue, ok := value.(string); !ok {
		return fmt.Errorf("bad value %v for %q attribute (expected string)", value, key)
	} else {
		*target = strValue
		return nil
	}
}
