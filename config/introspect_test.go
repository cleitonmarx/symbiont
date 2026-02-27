package config

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cleitonmarx/symbiont/introspection"
)

type simpleProvider struct {
	values map[string]string
	err    error
}

func (s simpleProvider) Get(ctx context.Context, key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	val, ok := s.values[key]
	if !ok {
		return "", errors.New("not found")
	}
	return val, nil
}

type providerWithName struct {
	values      map[string]string
	providerTag string
}

func (p providerWithName) Get(ctx context.Context, key string) (string, error) {
	val, ok := p.values[key]
	if !ok {
		return "", errors.New("not found")
	}
	return val, nil
}

func (p providerWithName) GetWithSource(ctx context.Context, key string) (string, string, error) {
	val, ok := p.values[key]
	if !ok {
		return "", p.providerTag, errors.New("not found")
	}
	return val, p.providerTag, nil
}

func stripLineAndOrderInfo(keys []introspection.ConfigAccess) []introspection.ConfigAccess {
	out := make([]introspection.ConfigAccess, len(keys))
	for i, k := range keys {
		k.Caller.Line = 0
		k.Order = 0
		out[i] = k
	}
	return out
}

func Test_providerInspector_get(t *testing.T) {
	tests := map[string]struct {
		providerValues map[string]string
		getKey         string
		wantValue      string
		wantErr        string
		wantKeys       []introspection.ConfigAccess
		repeatGet      bool
		wantHits       int
		withDefault    bool
	}{
		"records_key_on_success": {
			providerValues: map[string]string{"foo": "bar"},
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys: []introspection.ConfigAccess{
				{Key: "foo", Provider: "config.simpleProvider", UsedDefault: false, Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
			},
		},
		"does_not_record_on_error": {
			providerValues: map[string]string{},
			getKey:         "missing",
			wantValue:      "",
			wantErr:        "not found",
			wantKeys:       []introspection.ConfigAccess{},
		},
		"records_cache_hits": {
			providerValues: map[string]string{"foo": "bar"},
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: false, Key: "foo", Provider: "config.simpleProvider", Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
				{UsedDefault: false, Key: "foo", Provider: "config.simpleProvider", Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
			},
			repeatGet: true,
		},
		"records_key_with_default": {
			providerValues: map[string]string{},
			getKey:         "defaulted",
			wantValue:      "",
			wantErr:        "not found",
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: true, Key: "defaulted", Provider: "", Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
			},
			withDefault: true,
		},
		"records_key_with_default_and_cache": {
			providerValues: map[string]string{},
			getKey:         "defaulted2",
			wantValue:      "",
			wantErr:        "not found",
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: true, Key: "defaulted2", Provider: "", Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
				{UsedDefault: true, Key: "defaulted2", Provider: "", Caller: introspection.Caller{Func: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go"}},
			},
			repeatGet:   true,
			withDefault: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sp := simpleProvider{values: tt.providerValues}
			ip := newProviderInspector(sp)

			val, err := ip.get(context.Background(), tt.getKey, tt.withDefault, nil, 2)
			assertErrorMessage(t, err, tt.wantErr)
			if val != tt.wantValue {
				t.Fatalf("expected value %q, got %q", tt.wantValue, val)
			}

			if tt.repeatGet {
				val2, err2 := ip.get(context.Background(), tt.getKey, tt.withDefault, nil, 2)
				if val2 != tt.wantValue {
					t.Fatalf("expected repeated value %q, got %q", tt.wantValue, val2)
				}
				if err2 != nil {
					t.Fatalf("expected no error on repeated get, got %v", err2)
				}
			}

			keys := ip.getKeysAccessInfo()
			if !reflect.DeepEqual(stripLineAndOrderInfo(tt.wantKeys), stripLineAndOrderInfo(keys)) {
				t.Fatalf("expected keys %#v, got %#v", stripLineAndOrderInfo(tt.wantKeys), stripLineAndOrderInfo(keys))
			}
			for _, k := range keys {
				if k.Caller.Line <= 0 {
					t.Fatalf("expected caller line > 0, got %d", k.Caller.Line)
				}
				if k.Order <= 0 {
					t.Fatalf("expected order > 0, got %d", k.Order)
				}
			}
		})
	}
}

func Test_providerInspector_get_usingGetWithSource(t *testing.T) {
	tests := map[string]struct {
		providerValues map[string]string
		providerTag    string
		getKey         string
		wantValue      string
		wantErr        string
		wantKeys       []introspection.ConfigAccess
		repeatGet      bool
		defaultValue   bool
	}{
		"records_actual_provider": {
			providerValues: map[string]string{"foo": "bar"},
			providerTag:    "*config.providerWithName",
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys: []introspection.ConfigAccess{
				{Key: "foo", Provider: "*config.providerWithName", UsedDefault: false, Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
			},
		},
		"does_not_record_on_error": {
			providerValues: map[string]string{},
			providerTag:    "*config.providerWithName",
			getKey:         "missing",
			wantValue:      "",
			wantErr:        "not found",
			wantKeys:       []introspection.ConfigAccess{},
		},
		"records_cache_hits_with_provider": {
			providerValues: map[string]string{"foo": "bar"},
			providerTag:    "*config.providerWithName",
			getKey:         "foo",
			wantValue:      "bar",
			repeatGet:      true,
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: false, Key: "foo", Provider: "*config.providerWithName", Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
				{UsedDefault: false, Key: "foo", Provider: "*config.providerWithName", Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
			},
		},
		"records_with_empty_provider_tag": {
			providerValues: map[string]string{"empty": "val"},
			providerTag:    "",
			getKey:         "empty",
			wantValue:      "val",
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: false, Key: "empty", Provider: "", Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
			},
		},
		"records_with_provider_tag_and_default": {
			providerValues: map[string]string{},
			providerTag:    "*config.providerWithName",
			getKey:         "defaulted",
			wantValue:      "",
			wantErr:        "not found",
			wantKeys: []introspection.ConfigAccess{
				{UsedDefault: false, Key: "defaulted", Provider: "", Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
				{UsedDefault: true, Key: "defaulted", Provider: "", Caller: introspection.Caller{Func: "config.(*providerInspector).get", File: "config/introspect.go"}},
			},
			defaultValue: true,
			repeatGet:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			p := providerWithName{values: tt.providerValues, providerTag: tt.providerTag}
			ip := newProviderInspector(p)

			val, err := ip.get(context.Background(), tt.getKey, tt.defaultValue, nil, 1)
			assertErrorMessage(t, err, tt.wantErr)
			if val != tt.wantValue {
				t.Fatalf("expected value %q, got %q", tt.wantValue, val)
			}

			if tt.repeatGet {
				val, err := ip.get(context.Background(), tt.getKey, false, nil, 1)
				if val != tt.wantValue {
					t.Fatalf("expected repeated value %q, got %q", tt.wantValue, val)
				}
				if err != nil {
					t.Fatalf("expected no error on repeated get, got %v", err)
				}
			}

			keys := ip.getKeysAccessInfo()
			if !reflect.DeepEqual(stripLineAndOrderInfo(tt.wantKeys), stripLineAndOrderInfo(keys)) {
				t.Fatalf("expected keys %#v, got %#v", stripLineAndOrderInfo(tt.wantKeys), stripLineAndOrderInfo(keys))
			}
			for _, k := range keys {
				if k.Caller.Line <= 0 {
					t.Fatalf("expected caller line > 0, got %d", k.Caller.Line)
				}
				if k.Order <= 0 {
					t.Fatalf("expected order > 0, got %d", k.Order)
				}
			}
		})
	}
}

func Test_providerInspector_getKeysAccessInfo_sorted(t *testing.T) {
	sp := &simpleProvider{values: map[string]string{"b": "2", "a": "1"}}
	ip := newProviderInspector(sp)

	_, _ = ip.get(context.Background(), "b", false, nil, 1)
	_, _ = ip.get(context.Background(), "a", false, nil, 1)

	keys := ip.getKeysAccessInfo()
	if keys[0].Key != "a" {
		t.Fatalf("expected first key %q, got %q", "a", keys[0].Key)
	}
	if keys[1].Key != "b" {
		t.Fatalf("expected second key %q, got %q", "b", keys[1].Key)
	}
}
