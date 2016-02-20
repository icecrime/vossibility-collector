package transformation

import (
	"fmt"

	"cmd/vossibility-collector/config"
	"object/template"
)

// Transformations is a collection of Transformation for different event types.
type Transformations struct {
	funcs           template.FuncMap
	transformations map[string]Transformation
}

func NewTransformations() *Transformations {
	return &Transformations{
		transformations: make(map[string]Transformation),
	}
}

func (t *Transformations) Builtins() template.FuncMap {
	return template.FuncMap{
		"apply_transformation": t.fnApplyTransformation,
	}
}

func (t *Transformations) Funcs(funcs template.FuncMap) *Transformations {
	t.funcs = funcs
	return t
}

func (t *Transformations) Get(name string) Transformation {
	return t.transformations[name]
}

func (t *Transformations) Load(config config.SerializedTable) error {
	for event, def := range config {
		tr, err := TransformationFromConfig(def, t.funcs)
		if err != nil {
			return err
		}
		t.transformations[event] = tr
	}
	return nil
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
