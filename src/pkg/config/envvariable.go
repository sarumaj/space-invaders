package config

import (
	"reflect"
	"strconv"
	"strings"
	"sync"

	"encoding/json"
)

// envCache holds the already parsed environment variables, keyed by the raw
// declaration of the variable, including its fallback.
var (
	envCache      = make(map[string]any)
	envCacheMutex sync.RWMutex
)

// invalidateEnvCache drops every parsed environment value. It has to be called
// whenever the underlying environment changes, which is when it is refreshed
// from the server and when a variable is set or unset.
func invalidateEnvCache() {
	envCacheMutex.Lock()
	clear(envCache)
	envCacheMutex.Unlock()
}

// envVariable represents an environment variable.
type EnvVariable[T any] string

// Get returns the value of the environment variable.
//
// Several variables are read for every object on every frame, and a read costs
// a lookup in the JS environment object plus a reflection based parse, so the
// result is memoized until the environment changes.
func (e EnvVariable[T]) Get() (result T) {
	envCacheMutex.RLock()
	cached, cacheOk := envCache[string(e)]
	envCacheMutex.RUnlock()

	if cacheOk {
		if value, ok := cached.(T); ok {
			return value
		}
	}

	parse := func(raw string) (result T) {
		if len(raw) == 0 {
			return
		}

		target := reflect.ValueOf(&result).Elem()

		switch target.Kind() {
		case reflect.Bool:
			v, _ := strconv.ParseBool(raw)
			target.Set(reflect.ValueOf(v))
			return target.Interface().(T)

		case reflect.Float32, reflect.Float64:
			v, _ := strconv.ParseFloat(raw, 64)
			target.Set(reflect.ValueOf(v).Convert(target.Type()))
			return target.Interface().(T)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v, _ := strconv.ParseInt(raw, 10, 64)
			target.Set(reflect.ValueOf(v).Convert(target.Type()))
			return target.Interface().(T)

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v, _ := strconv.ParseUint(raw, 10, 64)
			target.Set(reflect.ValueOf(v).Convert(target.Type()))
			return target.Interface().(T)

		case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
			_ = json.Unmarshal([]byte(raw), target.Addr().Interface())
			return target.Interface().(T)

		case reflect.String:
			return any(raw).(T)

		default:
			return

		}
	}

	key, fallback, _ := strings.Cut(string(e), ":")
	raw := Getenv(key)
	if len(raw) == 0 && len(fallback) > 0 {
		raw = fallback
	}

	result = parse(raw)

	envCacheMutex.Lock()
	envCache[string(e)] = result
	envCacheMutex.Unlock()

	return
}
