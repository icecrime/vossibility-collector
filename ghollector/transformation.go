package main

import (
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/icecrime/vossibility/ghollector/template"
)

type visitor struct {
	values []interface{}
}

func (v *visitor) Value() interface{} {
	if len(v.values) == 1 {
		return v.values[0]
	}
	return v.values
}

func (v *visitor) Visit(i interface{}) {
	v.values = append(v.values, i)
}

// Transformation is the transformation matrix for a given payload.
type Transformation struct {
	event     string
	templates map[string]*template.Template
}

// NewTransformation creates an empty Transformation.
func NewTransformation(event string) *Transformation {
	return &Transformation{
		event:     event,
		templates: make(map[string]*template.Template),
	}
}

// TransformationFromConfig creates a transformation from a configuration
// description.
func TransformationFromConfig(event string, config map[string]string) (out *Transformation, err error) {
	out = NewTransformation(event)
	for key, tmpl := range config {
		var t *template.Template
		if tmpl != "" {
			if t, err = template.New(key).Parse(tmpl); err != nil {
				return nil, err
			}
		}
		out.templates[key] = t
	}
	return out, nil
}

// Apply takes a serialized JSON payload and returns a Blob on which the
// transformation has been applied, as well as a collection of metadata
// corresponding to fields prefixed by an underscore.
func (t Transformation) Apply(payload []byte) (*Blob, error) {
	sj, err := simplejson.NewJson(payload)
	if err != nil {
		return nil, err
	}

	m, err := sj.Map()
	if err != nil {
		return nil, err
	}

	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
	res := NewBlob(t.event)
	for key, tmpl := range t.templates {
		// A nil template is just a pass-through.
		if tmpl == nil {
			path := strings.Split(key, ".")
			v := sj.GetPath(path...).Interface()
			res.Push(key, v)
			continue
		}

		// Visit the template to extract the field values.
		vis := &visitor{}
		if err := tmpl.Execute(vis, m); err != nil {
			return nil, err
		}
		res.Push(key, vis.Value())
	}
	return res, nil
}

// ApplyBlob takes a serialized JSON payload and returns a Blob on which the
// transformation has been applied, as well as a collection of metadata
// corresponding to fields prefixed by an underscore.
func (t Transformation) ApplyBlob(b *Blob) (*Blob, error) {
	m, err := b.Data.Map()
	if err != nil {
		return nil, err
	}

	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
	res := NewBlob(b.Event)
	for key, tmpl := range t.templates {
		// A nil template is just a pass-through.
		if tmpl == nil {
			path := strings.Split(key, ".")
			v := b.Data.GetPath(path...).Interface()
			res.Push(key, v)
			continue
		}

		// Visit the template to extract the field values.
		vis := &visitor{}
		if err := tmpl.Execute(vis, m); err != nil {
			return nil, err
		}
		res.Push(key, vis.Value())
	}
	return res, nil
}
