package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
)

const (
	// MetadataTimestamp is the key for the timestamp metadata attribute used
	// when storing a Blob instance into the Elastic Search backend.
	MetadataTimestamp = "_timestamp"

	// MetadataType is the key for the type metadata attribute used when
	// storing a Blob instance into the Elastic Search backend.
	MetadataType = "_type"

	// MetadataSnapshotId is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the Id to be used when storing the snapshoted content of a
	// blob.
	MetadataSnapshotId = "_snapshot_id"

	// MetadataSnapshotField is the key for the snapshot id metadata attribute
	// used when storing a Blob instance into the Elastic Search backend. It
	// represents the nested object to be used when storing the snapshoted
	// content of a blob.
	MetadataSnapshotField = "_snapshot_field"
)

// NewBlob returns an empty Blob for that particular event type.
func NewBlob(event string) *Blob {
	return NewBlobFromJson(event, simplejson.New())
}

// Blob is a opaque type representing an arbitrary payload from GitHub.
type Blob struct {
	// Event is the type of event associated with this blob.
	Event string

	// Data is the payload content.
	Data *simplejson.Json

	// Metadata is a collection of (key, value) pairs that decorate the blob
	// and are used upon storage.
	Metadata map[string]interface{}
}

func NewBlobFromJson(event string, json *simplejson.Json) *Blob {
	return &Blob{
		Data:     json,
		Event:    event,
		Metadata: make(map[string]interface{}),
	}
}

func NewBlobFromPayload(event string, payload []byte) (*Blob, error) {
	d, err := simplejson.NewJson(payload)
	if err != nil {
		return nil, err
	}
	return NewBlobFromJson(event, d), nil
}

func (b *Blob) Encode() ([]byte, error) {
	return b.Data.Encode()
}

func (b *Blob) HasAttribute(attr string) bool {
	_, ok := b.Data.CheckGet(attr)
	return ok
}

func (b *Blob) Push(key string, value interface{}) {
	if strings.HasPrefix(key, "_") {
		b.Metadata[key] = value
		return
	}
	path := strings.Split(key, ".")
	b.Data.SetPath(path, value)
}

func (b *Blob) Timestamp() time.Time {
	if t, ok := b.Metadata[MetadataTimestamp]; ok {
		return t.(time.Time)
	}
	return time.Now()
}

func (b *Blob) Type() string {
	if t, ok := b.Metadata[MetadataType]; ok {
		return fmt.Sprintf("%v", t)
	}
	return b.Event
}

// Snapshot returns the Id and Data for the snapshot for a Blob that models a
// live event.
func (b *Blob) Snapshot() (string, *Blob) {
	if i, ok := b.Metadata[MetadataSnapshotId]; ok {
		if t, ok := b.Metadata[MetadataSnapshotField]; ok {
			nb := NewBlobFromJson(b.Event, b.Data.Get(t.(string)))
			ni := nb.Data.GetPath(strings.Split(i.(string), ".")...).MustInt()
			return strconv.Itoa(ni), nb
		}
	}
	return "", nil
}
