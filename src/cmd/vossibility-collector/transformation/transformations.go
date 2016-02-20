package transformation

import (
	"fmt"

	"cmd/vossibility-collector/config"
	"object/template"
)

// Transformations is a collection of Transformation for different event types.
type Transformations struct {
	Funcs           template.FuncMap
	transformations map[string]Transformation
}

func NewTransformations() *Transformations {
	t := &Transformations{
		transformations: make(map[string]Transformation),
	}
	t.Funcs = t.builtins()
	return t
}

func (t *Transformations) Get(name string) Transformation {
	return t.transformations[name]
}

func (t *Transformations) Load(config config.SerializedTable) error {
	for event, def := range config {
		tr, err := TransformationFromConfig(def, t.Funcs)
		if err != nil {
			return err
		}
		t.transformations[event] = tr
	}
	return nil
}

func (t *Transformations) builtins() template.FuncMap {
	return template.FuncMap{
		"apply_transformation": t.fnApplyTransformation,
	}
}

func (t *Transformations) fnApplyTransformation(name string, data interface{}) (interface{}, error) {
	f, ok := t.transformations[name]
	if !ok {
		return nil, fmt.Errorf("no such transformation %q", name)
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot apply transformation to non-object %v", data)
	}
	return f.ApplyMap(m)
}
