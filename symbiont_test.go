package symbiont

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper types used in tests
type recCloser struct {
	name string
	log  *[]string
}

func (r *recCloser) Initialize(ctx context.Context) (context.Context, error) { return ctx, nil }

func (r *recCloser) Close() { *r.log = append(*r.log, r.name) }

type ctxKeyType string

type ctxInitializer struct {
	key ctxKeyType
	val string
}

func (c ctxInitializer) Initialize(ctx context.Context) (context.Context, error) {
	return context.WithValue(ctx, c.key, c.val), nil
}

type errInitializer struct{}

func (errInitializer) Initialize(ctx context.Context) (context.Context, error) {
	return ctx, errors.New("init error")
}

type panicInitializer struct{}

func (panicInitializer) Initialize(context.Context) (context.Context, error) { panic("boom") }

// runnable that can return error, panic, read ctx key and/or record a closer name
type runCloser struct {
	name      string
	log       *[]string
	ctxKey    string
	gotVal    *string
	willErr   bool
	willPanic bool
}

func (r *runCloser) Run(ctx context.Context) error {
	if r.willPanic {
		panic("boom")
	}
	if r.willErr {
		return errors.New("run error")
	}
	if r.gotVal != nil && r.ctxKey != "" {
		if v := ctx.Value(ctxKeyType(r.ctxKey)); v != nil {
			*r.gotVal = v.(string)
		}
	}
	return nil
}
func (r *runCloser) Close() { *r.log = append(*r.log, r.name) }

// dependency-registering initializer used in tests
type depRegisterInitializer struct{ value string }

func (d *depRegisterInitializer) Initialize(ctx context.Context) (context.Context, error) {
	// register dependency for later resolution by runnables
	depend.ClearContainer()
	depend.Register(d.value)
	return ctx, nil
}

// runnable that receives a registered dependency via struct tag
// The field `Dep` is injected from the container using `resolve:""`.
type resolveDepRun struct {
	Dep    string `resolve:""`
	gotVal *string
}

func (r *resolveDepRun) Run(ctx context.Context) error {
	if r.gotVal != nil {
		*r.gotVal = r.Dep
	}
	return nil
}

// simple provider for config tests
type simpleProvider struct{ key, val string }

func (s *simpleProvider) Get(ctx context.Context, name string) (string, error) {
	if name == s.key {
		return s.val, nil
	}
	return "", errors.New("not found")
}

// initializer that sets the global provider
type setProviderInitializer struct{ key, val string }

func (s *setProviderInitializer) Initialize(ctx context.Context) (context.Context, error) {
	config.SetGlobalProvider(&simpleProvider{key: s.key, val: s.val})
	return ctx, nil
}

// runnable that receives a config value via struct tag
// The field `Cfg` is injected from the provider using `config:"cfgKey"`.
type configRun struct {
	Cfg    string `config:"cfgKey"`
	gotVal *string
}

func (c *configRun) Run(ctx context.Context) error {
	if c.gotVal != nil {
		*c.gotVal = c.Cfg
	}
	return nil
}

func TestApp_RunWithContext(t *testing.T) {
	type testCase struct {
		inits []Initializer
		runs  []Runnable
	}

	tests := map[string]struct {
		inits     []Initializer
		runs      []Runnable
		expectErr string
		validate  func(t *testing.T, tt *testCase, err error)
	}{
		"success_and_closers_lifo": {
			inits: []Initializer{
				&ctxInitializer{key: ctxKeyType("k"), val: "v"},
				&recCloser{name: "initA", log: &[]string{}},
				&recCloser{name: "initB", log: &[]string{}},
			},
			runs: []Runnable{
				&runCloser{name: "run1", log: &[]string{}, ctxKey: string(ctxKeyType("k")), gotVal: new(string)},
				&runCloser{name: "run2", log: &[]string{}, ctxKey: string(ctxKeyType("k")), gotVal: new(string)},
			},
			validate: func(t *testing.T, tt *testCase, err error) {
				assert.NoError(t, err)
				// Extract close log from first closer
				var closeLog []string
				for _, in := range tt.inits {
					if rc, ok := in.(*recCloser); ok {
						closeLog = *rc.log
						break
					}
				}
				assert.Equal(t, []string{"run2", "run1", "initB", "initA"}, closeLog)
				// Verify at least one runnable observed the context value
				found := false
				for _, r := range tt.runs {
					if rc, ok := r.(*runCloser); ok && rc.gotVal != nil && *rc.gotVal == "v" {
						found = true
						break
					}
				}
				assert.True(t, found, "no runnable received context value")
			},
		},
		"initializer_error": {
			inits:     []Initializer{&errInitializer{}},
			expectErr: "error: init error, component: symbiont.errInitializer",
			validate: func(t *testing.T, _ *testCase, err error) {
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "init error")
			},
		},
		"initializer_panic": {
			inits:     []Initializer{&panicInitializer{}},
			expectErr: "panic in setup func",
			validate: func(t *testing.T, _ *testCase, err error) {
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "panic in Initialize func: boom")
			},
		},
		"runnable_error": {
			runs: []Runnable{&runCloser{willErr: true}},
			validate: func(t *testing.T, _ *testCase, err error) {
				// runnable error propagates through errgroup
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "run error")
			},
		},
		"runnable_panic": {
			runs:      []Runnable{&runCloser{willPanic: true}},
			expectErr: "panic in Run func",
			validate: func(t *testing.T, _ *testCase, err error) {
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "panic in Run func: boom")
			},
		},
		"init_registers_dependency_and_runner_resolves": {
			inits: []Initializer{&depRegisterInitializer{value: "dep-val"}},
			runs:  []Runnable{&resolveDepRun{gotVal: new(string)}},
			validate: func(t *testing.T, tt *testCase, err error) {
				assert.NoError(t, err)
				if rr, ok := tt.runs[0].(*resolveDepRun); ok {
					assert.Equal(t, "dep-val", *rr.gotVal)
				}
			},
		},
		"init_sets_provider_and_runner_reads_config": {
			inits: []Initializer{&setProviderInitializer{key: "cfgKey", val: "cfgVal"}},
			runs:  []Runnable{&configRun{gotVal: new(string)}},
			validate: func(t *testing.T, tt *testCase, err error) {
				assert.NoError(t, err)
				if cr, ok := tt.runs[0].(*configRun); ok {
					assert.Equal(t, "cfgVal", *cr.gotVal)
				}
			},
		},
		"resolve_missing_dependency": {
			runs:      []Runnable{&resolveDepRun{gotVal: new(string)}},
			expectErr: "the dependency type 'string' was not registered",
			validate: func(t *testing.T, _ *testCase, err error) {
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "not registered")
			},
		},
		"config_missing_key": {
			inits:     []Initializer{&setProviderInitializer{key: "otherKey", val: "x"}},
			runs:      []Runnable{&configRun{gotVal: new(string)}},
			expectErr: "error getting value for field",
			validate: func(t *testing.T, _ *testCase, err error) {
				var se Error
				require.Error(t, err)
				require.True(t, errors.As(err, &se))
				assert.Contains(t, se.Error(), "not found")
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			depend.ClearContainer()
			config.ResetGlobalProvider()
			defer func() { depend.ClearContainer(); config.ResetGlobalProvider() }()

			// Setup: attach shared close log to all closers
			closeLog := []string{}
			for i, in := range test.inits {
				if rc, ok := in.(*recCloser); ok {
					rc.log = &closeLog
					test.inits[i] = rc
				}
			}
			for i, r := range test.runs {
				if rc, ok := r.(*runCloser); ok {
					rc.log = &closeLog
					test.runs[i] = rc
				}
			}

			// Build app
			a := NewApp()
			for _, in := range test.inits {
				a.Initialize(in)
			}
			for _, r := range test.runs {
				a.Host(r)
			}

			// Run and validate
			err := a.RunWithContext(context.Background())
			test.validate(t, &testCase{inits: test.inits, runs: test.runs}, err)
		})
	}
}

// waitRunnable blocks until the app context is cancelled.
type waitRunnable struct{ done chan struct{} }

func (w *waitRunnable) Run(ctx context.Context) error { <-ctx.Done(); close(w.done); return nil }

func TestApp_Run_StopsOnInterrupt(t *testing.T) {
	// isolate global state
	depend.ClearContainer()
	config.ResetGlobalProvider()
	defer func() { depend.ClearContainer(); config.ResetGlobalProvider() }()

	a := NewApp()
	w := &waitRunnable{done: make(chan struct{})}
	a.Host(w)

	// run in background and send an interrupt
	errCh := make(chan error, 1)
	go func() { errCh <- a.Run() }()
	// allow goroutine to start
	time.Sleep(10 * time.Millisecond)
	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(os.Interrupt)

	select {
	case <-w.done:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("waitRunnable did not stop after interrupt")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return after interrupt")
	}
}

func TestApp_RunAsync_ContextCancel(t *testing.T) {
	// isolate global state
	depend.ClearContainer()
	config.ResetGlobalProvider()
	defer func() { depend.ClearContainer(); config.ResetGlobalProvider() }()

	a := NewApp()
	w := &waitRunnable{done: make(chan struct{})}
	a.Host(w)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := a.RunAsync(ctx)
	// allow goroutine to start
	err := a.WaitForReadiness(ctx, 500*time.Millisecond)
	assert.NoError(t, err)
	// cancel the context to stop the app
	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("RunAsync did not return after context cancel")
	}

	select {
	case <-w.done:
		// ok
	case <-time.After(500 * time.Millisecond):
		t.Fatal("waitRunnable did not stop after context cancel")
	}
}
