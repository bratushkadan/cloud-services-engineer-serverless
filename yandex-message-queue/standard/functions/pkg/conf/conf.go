package conf

import (
	"fmt"
	"os"
)

func MustEnv(envKey string) string {
	env := os.Getenv(envKey)
	if env == "" {
		panic(fmt.Sprintf(`Env "%s" must be set.`, envKey))
	}
	return env
}
