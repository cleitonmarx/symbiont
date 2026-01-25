package symbiont

import (
	"context"
	"testing"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
)

type mapProvider struct {
	values map[string]string
}

func (m mapProvider) Get(ctx context.Context, name string) (string, error) {
	if v, ok := m.values[name]; ok {
		return v, nil
	}
	return "", assert.AnError
}

type initForIntrospect struct{}

func (initForIntrospect) Initialize(ctx context.Context) (context.Context, error) {
	depend.ClearContainer()
	depend.Register("depVal")
	_, err := config.Get[string](ctx, "cfgKey")
	return ctx, err
}

type runForIntrospect struct {
	Dep string `resolve:""`
}

func (r *runForIntrospect) Run(ctx context.Context) error { return nil }

type recorderIntrospector struct {
	report    introspection.Report
	called    bool
	willErr   bool
	willPanic bool
}

func (r *recorderIntrospector) Introspect(_ context.Context, rep introspection.Report) error {
	if r.willPanic {
		panic("introspector panic")
	}
	r.report = rep
	r.called = true
	if r.willErr {
		return assert.AnError
	}
	return nil
}

func TestApp_IntrospectProvidesReport(t *testing.T) {
	type tc struct {
		name      string
		intro     *recorderIntrospector
		expectErr bool
		expectPan bool
	}

	cases := []tc{
		{
			name:      "success",
			intro:     &recorderIntrospector{},
			expectErr: false,
			expectPan: false,
		},
		{
			name:      "introspector-returns-error",
			intro:     &recorderIntrospector{willErr: true},
			expectErr: true,
			expectPan: false,
		},
		{
			name:      "introspector-panics",
			intro:     &recorderIntrospector{willPanic: true},
			expectErr: false,
			expectPan: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer depend.ClearContainer()
			defer config.ResetGlobalProvider()
			config.SetGlobalProvider(mapProvider{values: map[string]string{"cfgKey": "val"}})

			app := NewApp().
				Initialize(&initForIntrospect{}).
				Host(&runForIntrospect{}).
				Introspect(c.intro)

			err := app.RunWithContext(context.Background())
			if c.expectPan {
				assert.Error(t, err, "expected error when introspector panics")
				return
			}
			if c.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, c.intro.called, "introspector should be invoked")
				assert.Len(t, c.intro.report.Initializers, 1)
				assert.Contains(t, c.intro.report.Initializers[0].Type, "initForIntrospect")
			}
		})
	}
}
