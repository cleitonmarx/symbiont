package symbiont

import (
	"context"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	report introspection.Report
	called bool
}

func (r *recorderIntrospector) Introspect(_ context.Context, rep introspection.Report) error {
	r.report = rep
	r.called = true
	return nil
}

func TestApp_IntrospectProvidesReport(t *testing.T) {
	defer depend.ClearContainer()
	defer config.ResetGlobalProvider()
	config.SetGlobalProvider(mapProvider{values: map[string]string{"cfgKey": "val"}})

	ri := &recorderIntrospector{}
	app := NewApp().
		Initialize(&initForIntrospect{}).
		Host(&runForIntrospect{}).
		Instrospect(ri)

	require.NoError(t, app.RunWithContext(context.Background()))
	require.True(t, ri.called, "introspector should be invoked")

	assert.Len(t, ri.report.Initializers, 1)
	assert.True(t, strings.Contains(ri.report.Initializers[0].Type, "initForIntrospect"))

	assert.Len(t, ri.report.Runners, 1)
	assert.True(t, strings.Contains(ri.report.Runners[0].Type, "runForIntrospect"))

	if assert.Len(t, ri.report.Configs, 1) {
		cfg := ri.report.Configs[0]
		assert.Equal(t, "cfgKey", cfg.Key)
		assert.False(t, cfg.UsedDefault)
	}

	assert.GreaterOrEqual(t, len(ri.report.Deps), 1)
	hasRegister := false
	for _, d := range ri.report.Deps {
		if d.Kind == introspection.DepRegistered {
			hasRegister = true
			assert.Equal(t, "string", d.Impl)
			assert.True(t, strings.Contains(d.Caller.Func, "initForIntrospect"))
		}
	}
	assert.True(t, hasRegister, "expected a registered dependency in report")
}
