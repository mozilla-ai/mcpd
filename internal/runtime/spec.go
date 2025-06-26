package runtime

import (
	"fmt"
	"strings"
)

type Spec struct {
	ShouldIgnoreFlag   func(string) bool
	ExtractPackageName func([]string) (string, error)
}

func Specs() map[Runtime]Spec {
	return map[Runtime]Spec{
		Docker: {
			ShouldIgnoreFlag: func(flag string) bool {
				switch flag {
				case "--rm", "--name", "--volume", "-v", "--network", "--detach", "-d", "-i":
					return true
				}
				return false
			},
			ExtractPackageName: func(args []string) (string, error) {
				skip := true
				skipNext := false

				// Flags that take a value (e.g. --name greptime)
				flagsWithValues := map[string]struct{}{
					"-e":        {},
					"--env":     {},
					"-p":        {},
					"-v":        {},
					"--volume":  {},
					"--name":    {},
					"--network": {},
				}

				for i := 0; i < len(args); i++ {
					arg := args[i]

					if skip {
						if arg == "run" {
							skip = false
						}
						continue
					}

					if skipNext {
						skipNext = false
						continue
					}

					if strings.HasPrefix(arg, "-") {
						_, takesValue := flagsWithValues[arg]
						skipNext = takesValue
						continue
					}

					return arg, nil
				}

				return "", fmt.Errorf("no %s image found", Docker)
			},
		},
		NPX: {
			ShouldIgnoreFlag: func(flag string) bool {
				return flag == "-y"
			},
			ExtractPackageName: func(args []string) (string, error) {
				// First non-flag value
				for _, arg := range args {
					if !strings.HasPrefix(arg, "-") {
						return arg, nil
					}
				}
				return "", fmt.Errorf("no %s package found", NPX)
			},
		},
		UVX: {
			ShouldIgnoreFlag: func(flag string) bool {
				return false
			},
			ExtractPackageName: func(args []string) (string, error) {
				for _, arg := range args {
					if !strings.HasPrefix(arg, "-") {
						return arg, nil
					}
				}
				return "", fmt.Errorf("no %s binary found", UVX)
			},
		},
		Python: {
			ShouldIgnoreFlag: func(flag string) bool {
				return flag == "-m"
			},
			ExtractPackageName: func(args []string) (string, error) {
				// No clear package name? Could return empty or static
				return "", nil
			},
		},
	}
}
