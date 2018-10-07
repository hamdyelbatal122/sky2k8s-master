package main

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
)

// load is a simple sequential implementation of module loading.
func load(thread *skylark.Thread, module string) (skylark.StringDict, error) {
	type entry struct {
		globals skylark.StringDict
		err     error
	}

	cache := make(map[string]*entry)
	e, ok := cache[module]
	if e == nil {
		if ok {
			// request for package whose loading is in progress
			return nil, fmt.Errorf("cycle in load graph")
		}

		// Add a placeholder to indicate "load in progress".
		cache[module] = nil

		// Load it.
		thread := &skylark.Thread{Load: load}
		err := skylark.ExecFile(thread, module, nil, globals)
		e = &entry{globals, err}

		// Update the cache.
		cache[module] = e
	}
	return e.globals, e.err
}

type k8sObject struct {
	ApiVersion skylark.String         `json:"apiVersion"`
	Kind       skylark.String         `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
}

func convertDict(d *skylark.Dict) map[string]interface{} {
	r := map[string]interface{}{}

	for _, k := range d.Keys() {
		key, _ := skylark.AsString(k)
		v, _, _ := d.Get(k)
		r[key] = convert(v)
	}
	return r
}

func convertArray(l *skylark.List) []interface{} {
	elems := []interface{}{}
	for i := 0; i < l.Len(); i++ {
		v := l.Index(i)
		elems = append(elems, convert(v))
	}
	return elems
}

func convert(v skylark.Value) interface{} {
	switch v.(type) {
	case *skylark.Dict:
		d, _ := v.(*skylark.Dict)
		return convertDict(d)
	case *skylark.List:
		l, _ := v.(*skylark.List)
		return convertArray(l)
	case skylark.Int:
		i, _ := skylark.AsInt32(v)
		return i
	case skylark.Float:
		f, _ := skylark.AsFloat(v)
		return f
	default:
		return v
	}
}

func output_type(t *skylark.Thread, b *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {

	var s, m *skylark.Dict
	for _, kw := range kwargs {
		st, _ := skylark.AsString(kw[0])
		if st == "spec" {
			s, _ = kw[1].(*skylark.Dict)
		} else if st == "metadata" {
			m, _ = kw[1].(*skylark.Dict)
		}
	}

	apiVersion, _ := args[0].(skylark.String)
	kind, _ := args[1].(skylark.String)
	obj := k8sObject{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata:   convertDict(m),
		Spec:       convertDict(s),
	}

	by, err := yaml.Marshal(obj)
	if err != nil {
		return skylark.None, err
	}

	fmt.Println(string(by))

	return skylark.None, nil
}

var globals = skylark.StringDict{
	"output_type": skylark.NewBuiltin("output_type", output_type),
}

func main() {
	resolve.AllowFloat = true
	resolve.AllowLambda = true
	thread := &skylark.Thread{Load: load}

	if err := skylark.ExecFile(thread, os.Args[1], nil, globals); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
