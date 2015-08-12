package main

import (
	"fmt"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/icecrime/vossibility/ghollector/template"
)

type visitor struct {
	values []interface{}
}

func (v *visitor) Value() interface{} {
	switch len(v.values) {
	case 0:
		return nil
	case 1:
		return v.values[0]
	default:
		return v.values
	}
}

func (v *visitor) Visit(i interface{}) {
	v.values = append(v.values, i)
}

// Transformations is a collection of Transformation for different event types.
type Transformations map[string]*Transformation

// TransformationsFromConfig creates a transformation from a flat textual
// configuration description.
func TransformationsFromConfig(config map[string]map[string]string) (Transformations, error) {
	res := Transformations(make(map[string]*Transformation))
	funcs := template.FuncMap{
		"apply_transformation": res.fnApplyTransformation,
		"user_data":            res.fnApplyUserData,
	}
	for event, def := range config {
		t, err := TransformationFromConfig(event, def, funcs)
		if err != nil {
			return nil, err
		}
		res[event] = t
	}
	return res, nil
}

func (t Transformations) fnApplyUserData(login string) interface{} {
	// Ignore any error to retrieve the user data: we don't have entries for
	// most of our users, and only store information for those who have
	// particular status (employees and/or maintainers).
	us := &userStore{}
	if ud, err := us.Get(login); err == nil {
		return ud
	}
	return &UserData{Login: login}
}

func (t Transformations) fnApplyTransformation(name string, data interface{}) (interface{}, error) {
	f, ok := t[name]
	if !ok {
		return nil, fmt.Errorf("no such transformation %q", name)
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot apply transformation to non-object %v", data)
	}
	return f.ApplyMap(m)
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
func TransformationFromConfig(event string, config map[string]string, funcs template.FuncMap) (out *Transformation, err error) {
	out = NewTransformation(event)
	for key, tmpl := range config {
		var t *template.Template
		if t, err = template.New(key).Funcs(funcs).Parse(tmpl); err != nil {
			return nil, err
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

	// Create the result blob, but inherit from the parent's metadata.
	res := NewBlob(b.Event)
	for k, v := range b.Metadata {
		res.Metadata[k] = v
	}

	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
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

// ApplyMap is a less capable version of ApplyBlob that only knows how to deal
// with simple objects, and won't handle any metadata fields. It is used when
// applying a transformation to a nested object where metadata transformation
// is not expected.
func (t Transformation) ApplyMap(m map[string]interface{}) (map[string]interface{}, error) {
	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
	res := make(map[string]interface{})
	for key, tmpl := range t.templates {
		// A nil template is just a pass-through.
		if tmpl == nil {
			var v interface{}
			v = m
			for _, p := range strings.Split(key, ".") {
				m, ok := v.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid path %q in %v", key, m)
				}
				v = m[p]
			}
			res[key] = v
			continue
		}

		// Visit the template to extract the field values.
		vis := &visitor{}
		if err := tmpl.Execute(vis, m); err != nil {
			return nil, err
		}
		res[key] = vis.Value()
	}
	return res, nil
}
