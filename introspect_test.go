package symbiont

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/introspection"
)

var errTest = errors.New("test error")

type mapProvider struct {
	values map[string]string
}

func (m mapProvider) Get(ctx context.Context, name string) (string, error) {
	if v, ok := m.values[name]; ok {
		return v, nil
	}
	return "", errTest
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

type runnableIntrospector struct {
	report           introspection.Report
	introspectCalled bool
	runCalled        bool
}

func (r *runnableIntrospector) Run(_ context.Context) error {
	r.runCalled = true
	return nil
}

func (r *runnableIntrospector) Introspect(_ context.Context, rep introspection.Report) error {
	r.report = rep
	r.introspectCalled = true
	return nil
}

type runnableOnlyIntrospector struct {
	Dep              string `resolve:""`
	Cfg              string `config:"cfgKey"`
	report           introspection.Report
	introspectCalled bool
	runCalled        bool
}

func (r *runnableOnlyIntrospector) Run(_ context.Context) error {
	r.runCalled = true
	return nil
}

func (r *runnableOnlyIntrospector) Introspect(_ context.Context, rep introspection.Report) error {
	r.report = rep
	r.introspectCalled = true
	return nil
}

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
		return errTest
	}
	return nil
}

func TestApp_IntrospectProvidesReport(t *testing.T) {
	type tc struct {
		name      string
		intro     Introspector
		register  func(*App)
		host      Runnable
		expectErr bool
		expectPan bool
		validate  func(t *testing.T, intro Introspector, hosted Runnable)
	}

	cases := []tc{
		{
			name:      "success",
			intro:     &recorderIntrospector{},
			host:      &runForIntrospect{},
			expectErr: false,
			expectPan: false,
		},
		{
			name:      "introspector-returns-error",
			intro:     &recorderIntrospector{willErr: true},
			host:      &runForIntrospect{},
			expectErr: true,
			expectPan: false,
		},
		{
			name:      "introspector-panics",
			intro:     &recorderIntrospector{willPanic: true},
			host:      &runForIntrospect{},
			expectErr: false,
			expectPan: true,
		},
		{
			name:      "hosted-runnable-also-introspector",
			host:      &runnableIntrospector{},
			expectErr: false,
			expectPan: false,
			validate: func(t *testing.T, _ Introspector, hosted Runnable) {
				ri, ok := hosted.(*runnableIntrospector)
				if !ok {
					t.Fatalf("expected hosted runnable to be *runnableIntrospector")
				}
				if !ri.introspectCalled {
					t.Fatal("hosted runnable introspector should be invoked")
				}
				if !ri.runCalled {
					t.Fatal("hosted runnable should still run")
				}
				if len(ri.report.Runners) != 1 {
					t.Fatalf("expected 1 runner, got %d", len(ri.report.Runners))
				}
				if !strings.Contains(ri.report.Runners[0].Type, "runnableIntrospector") {
					t.Fatalf("expected runner type to contain runnableIntrospector, got %q", ri.report.Runners[0].Type)
				}
				if len(ri.report.Initializers) != 1 {
					t.Fatalf("expected 1 initializer, got %d", len(ri.report.Initializers))
				}
				if !strings.Contains(ri.report.Initializers[0].Type, "initForIntrospect") {
					t.Fatalf("expected initializer type to contain initForIntrospect, got %q", ri.report.Initializers[0].Type)
				}
			},
		},
		{
			name:      "introspect-only-runnable-is-wired",
			intro:     &runnableOnlyIntrospector{},
			host:      &runForIntrospect{},
			expectErr: false,
			expectPan: false,
			validate: func(t *testing.T, intro Introspector, _ Runnable) {
				ri, ok := intro.(*runnableOnlyIntrospector)
				if !ok {
					t.Fatalf("expected introspector to be *runnableOnlyIntrospector")
				}
				if !ri.introspectCalled {
					t.Fatal("introspector should be invoked")
				}
				if ri.runCalled {
					t.Fatal("introspector should not run when only registered via Introspect")
				}
				if ri.Dep != "depVal" {
					t.Fatalf("expected resolved dependency %q, got %q", "depVal", ri.Dep)
				}
				if ri.Cfg != "val" {
					t.Fatalf("expected config value %q, got %q", "val", ri.Cfg)
				}
			},
		},
		{
			name: "nil-introspector-is-ignored",
			register: func(app *App) {
				app.Introspect(nil)
			},
			host:      &runForIntrospect{},
			expectErr: false,
			expectPan: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer depend.ClearContainer()
			defer config.ResetGlobalProvider()
			config.SetGlobalProvider(mapProvider{values: map[string]string{"cfgKey": "val"}})

			hosted := c.host
			if hosted == nil {
				hosted = &runForIntrospect{}
			}

			app := NewApp().
				Initialize(&initForIntrospect{}).
				Host(hosted)
			if c.register != nil {
				c.register(app)
			} else if c.intro != nil {
				app.Introspect(c.intro)
			}

			err := app.RunWithContext(context.Background())
			if c.expectPan {
				if err == nil {
					t.Fatal("expected error when introspector panics")
				}
				return
			}
			if c.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if intro, ok := c.intro.(*recorderIntrospector); ok {
					if !intro.called {
						t.Fatal("introspector should be invoked")
					}
					if len(intro.report.Initializers) != 1 {
						t.Fatalf("expected 1 initializer, got %d", len(intro.report.Initializers))
					}
					if !strings.Contains(intro.report.Initializers[0].Type, "initForIntrospect") {
						t.Fatalf("expected initializer type to contain initForIntrospect, got %q", intro.report.Initializers[0].Type)
					}
				}
				if c.validate != nil {
					c.validate(t, c.intro, hosted)
				}
			}
		})
	}
}
