package transformation

import (
	"fmt"
	"strings"

	"cmd/vossibility-collector/blob"
	"object/template"
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

// Transformation is the transformation matrix for a given payload.
type Transformation map[string]*template.Template

// NewTransformation creates an empty Transformation.
func NewTransformation() Transformation {
	return make(map[string]*template.Template)
}

// TransformationFromConfig creates a transformation from a configuration
// description.
func TransformationFromConfig(config map[string]string, funcs template.FuncMap) (out Transformation, err error) {
	out = NewTransformation()
	for key, tmpl := range config {
		var t *template.Template
		if t, err = template.New(key).Funcs(funcs).Parse(tmpl); err != nil {
			return nil, err
		}
		out[key] = t
	}
	return out, nil
}

// Apply takes a serialized JSON payload and returns a Blob on which the
// transformation has been applied, as well as a collection of metadata
// corresponding to fields prefixed by an underscore.
func (t Transformation) Apply(b *blob.Blob) (*blob.Blob, error) {
	m, err := b.Data.Map()
	if err != nil {
		return nil, err
	}

	// Create the result blob, but inherit from the parent's metadata.
	res := blob.NewBlob(b.Type, b.ID)
	res.Timestamp = b.Timestamp

	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
	for key, tmpl := range t {
		// Visit the template to extract the field values.
		vis := &visitor{}
		if err := tmpl.Execute(vis, m); err != nil {
			return nil, err
		}
		res.Push(key, vis.Value())
	}
	return res, nil
}

// ApplyMap is a less capable version of Apply that only knows how to deal with
// simple objects, and won't handle any metadata fields. It is used when
// applying a transformation to a nested object where metadata transformation
// is not expected.
func (t Transformation) ApplyMap(m map[string]interface{}) (map[string]interface{}, error) {
	// For each destination field defined in the transformation, apply the
	// associated template and store it in the output.
	res := make(map[string]interface{})
	for key, tmpl := range t {
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
