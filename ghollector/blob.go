package main

import (
	"fmt"
	"strings"

	"github.com/bitly/go-simplejson"
)

type Blob struct {
	Data     *simplejson.Json
	Event    string
	Metadata map[string]interface{}
}

func NewBlob(event string) *Blob {
	return &Blob{
		Data:     simplejson.New(),
		Event:    event,
		Metadata: make(map[string]interface{}),
	}
}

func NewBlobFromPayload(event string, payload []byte) (*Blob, error) {
	d, err := simplejson.NewJson(payload)
	if err != nil {
		return nil, err
	}
	b := NewBlob(event)
	b.Data = d
	return b, nil
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

func (b *Blob) Type() string {
	if t, ok := b.Metadata["_type"]; ok {
		return fmt.Sprintf("%v", t)
	}
	return b.Event
}

// Snapshot returns the Id and Data for the snapshot for a Blob that models a
// live event.
func (b *Blob) Snapshot() (string, *Blob) {
	if i, ok := b.Metadata["_snapshot_id"]; ok {
		if t, ok := b.Metadata["_snapshot_field"]; ok {
			nb := &Blob{
				Data:  b.Data.Get(t.(string)),
				Event: b.Event,
			}
			return i.(string), nb
		}
	}
	return "", nil
}
