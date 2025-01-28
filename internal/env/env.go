package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

type value interface {
	string | bool
}

func Lookup[T value](name string) (out T, err error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return out, fmt.Errorf("environment variable %s not found", name)
	}

	var tmp any
	switch any(out).(type) {
	case bool:
		tmp, err = strconv.ParseBool(v)
		if err != nil {
			return out, err
		}
	default:
		tmp = v
	}

	if t, ok := tmp.(T); ok {
		return t, nil
	}

	return out, fmt.Errorf("unable to parse %v as %T", tmp, out)
}
