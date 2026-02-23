package adapter

import "fmt"

var builtins = map[string]func() Adapter{}

func register(name string, factory func() Adapter) {
	builtins[name] = factory
}

func Get(name string) (Adapter, error) {
	factory, ok := builtins[name]
	if !ok {
		return nil, fmt.Errorf("unknown adapter: %q", name)
	}
	return factory(), nil
}

func BuiltinNames() []string {
	names := make([]string, 0, len(builtins))
	for name := range builtins {
		names = append(names, name)
	}
	return names
}
