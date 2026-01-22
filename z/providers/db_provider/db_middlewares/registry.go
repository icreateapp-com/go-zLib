package db_middlewares

import (
	"errors"
	"strings"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

type NamedMiddleware struct {
	Name string
	New  func() Middleware
}

type RegistryIn struct {
	fx.In
	Items []NamedMiddleware `group:"db_named_middlewares"`
}

type Registry struct {
	m map[string]func() Middleware
}

func NewRegistry(in RegistryIn) *Registry {
	r := &Registry{m: map[string]func() Middleware{}}
	for _, it := range in.Items {
		name := strings.TrimSpace(strings.ToLower(it.Name))
		if name == "" || it.New == nil {
			continue
		}
		r.m[name] = it.New
	}
	return r
}

func (r *Registry) Get(name string) (Middleware, bool) {
	if r == nil {
		return nil, false
	}
	fn, ok := r.m[strings.TrimSpace(strings.ToLower(name))]
	if !ok || fn == nil {
		return nil, false
	}
	mw := fn()
	if mw == nil {
		return nil, false
	}
	return mw, true
}

func (r *Registry) Apply(db *gorm.DB, names []string) error {
	for _, name := range names {
		mw, ok := r.Get(name)
		if !ok {
			return errors.New("unknown db middleware: " + name)
		}
		if err := mw(db); err != nil {
			return err
		}
	}
	return nil
}

var RegistryModule = fx.Options(
	fx.Provide(NewRegistry),
)
