package config

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

func Test_providerInspector_get(t *testing.T) {
	tests := map[string]struct {
		providerValues map[string]string
		getKey         string
		wantValue      string
		wantErr        error
		wantKeys       []KeyAccessInfo
		repeatGet      bool
		wantHits       int
		withDefault    bool
	}{
		"records_key_on_success": {
			providerValues: map[string]string{"foo": "bar"},
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys: []KeyAccessInfo{
				{Key: "foo", Provider: "config.simpleProvider", Default: false, Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 112},
			},
			repeatGet: false,
		},
		"does_not_record_on_error": {
			providerValues: map[string]string{},
			getKey:         "missing",
			wantValue:      "",
			wantErr:        errors.New("not found"),
			wantKeys:       []KeyAccessInfo{},
			repeatGet:      false,
		},
		"records_cache_hits": {
			providerValues: map[string]string{"foo": "bar"},
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys: []KeyAccessInfo{
				{Default: false, Key: "foo", Provider: "config.simpleProvider", Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 112},
				{Default: false, Key: "foo", Provider: "config.simpleProvider", Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 118}},
			repeatGet: true,
		},
		"records_key_with_default": {
			providerValues: map[string]string{},
			getKey:         "defaulted",
			wantValue:      "",
			wantErr:        errors.New("not found"),
			wantKeys:       []KeyAccessInfo{{Default: true, Key: "defaulted", Provider: "", Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 112}},
			withDefault:    true,
		},
		"records_key_with_default_and_cache": {
			providerValues: map[string]string{},
			getKey:         "defaulted2",
			wantValue:      "",
			wantErr:        errors.New("not found"),
			wantKeys: []KeyAccessInfo{
				{Default: true, Key: "defaulted2", Provider: "", Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 112},
				{Default: true, Key: "defaulted2", Provider: "", Caller: "config.Test_providerInspector_get.func1", File: "config/introspect_test.go", Line: 118},
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
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantValue, val)

			// If repeatGet is set, call Get again to test hits
			if tt.repeatGet {
				val2, err2 := ip.get(context.Background(), tt.getKey, tt.withDefault, nil, 2)
				assert.Equal(t, tt.wantValue, val2)
				assert.NoError(t, err2)
			}

			keys := ip.getKeysAccessInfo()
			assert.Equal(t, tt.wantKeys, keys)
		})
	}
}

func Test_providerInspector_get_usingGetWithSource(t *testing.T) {
	tests := map[string]struct {
		providerValues map[string]string
		providerTag    string
		getKey         string
		wantValue      string
		wantErr        error
		wantKeys       []KeyAccessInfo
		repeatGet      bool
		defaultValue   bool
	}{
		"records_actual_provider": {
			providerValues: map[string]string{"foo": "bar"},
			providerTag:    "*config.providerWithName",
			getKey:         "foo",
			wantValue:      "bar",
			wantKeys:       []KeyAccessInfo{{Key: "foo", Provider: "*config.providerWithName", Default: false, Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 97}},
		},
		"does_not_record_on_error": {
			providerValues: map[string]string{},
			providerTag:    "*config.providerWithName",
			getKey:         "missing",
			wantValue:      "",
			wantErr:        errors.New("not found"),
			wantKeys:       []KeyAccessInfo{},
		},
		"records_cache_hits_with_provider": {
			providerValues: map[string]string{"foo": "bar"},
			providerTag:    "*config.providerWithName",
			getKey:         "foo",
			wantValue:      "bar",
			repeatGet:      true,
			wantKeys: []KeyAccessInfo{
				{Default: false, Key: "foo", Provider: "*config.providerWithName", Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 79},
				{Default: false, Key: "foo", Provider: "*config.providerWithName", Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 97},
			},
		},
		"records_with_empty_provider_tag": {
			providerValues: map[string]string{"empty": "val"},
			providerTag:    "",
			getKey:         "empty",
			wantValue:      "val",
			wantKeys:       []KeyAccessInfo{{Default: false, Key: "empty", Provider: "", Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 97}},
		},
		"records_with_provider_tag_and_default": {
			providerValues: map[string]string{},
			providerTag:    "*config.providerWithName",
			getKey:         "defaulted",
			wantValue:      "",
			wantErr:        errors.New("not found"),
			wantKeys: []KeyAccessInfo{
				{Default: false, Key: "defaulted", Provider: "", Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 79},
				{Default: true, Key: "defaulted", Provider: "", Caller: "config.(*providerInspector).get", File: "config/introspect.go", Line: 97},
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
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantValue, val)

			// If repeatGet is set, call Get again to test hits
			if tt.repeatGet {
				val, err := ip.get(context.Background(), tt.getKey, false, nil, 1)
				assert.Equal(t, tt.wantValue, val)
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantKeys, ip.getKeysAccessInfo())
		})
	}
}

func Test_providerInspector_getKeysAccessInfo_sorted(t *testing.T) {
	sp := &simpleProvider{values: map[string]string{"b": "2", "a": "1"}}
	ip := newProviderInspector(sp)

	_, _ = ip.get(context.Background(), "b", false, nil, 1)
	_, _ = ip.get(context.Background(), "a", false, nil, 1)

	keys := ip.getKeysAccessInfo()
	assert.Equal(t, "a", keys[0].Key)
	assert.Equal(t, "b", keys[1].Key)
}
