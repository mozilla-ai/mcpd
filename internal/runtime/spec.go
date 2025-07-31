package runtime

import (
	"fmt"
	"strings"
)

type pkgExtractor func(args []string) (string, error)

type flagNameSet map[string]struct{}

type Spec struct {
	ShouldIgnoreFlag   func(string) bool
	ExtractPackageName func([]string) (string, error)
}

func NewSpec(ignoreFlags flagNameSet, extractor pkgExtractor) Spec {
	return Spec{
		ShouldIgnoreFlag: func(flagName string) bool {
			_, ok := ignoreFlags[flagName]
			return ok
		},
		ExtractPackageName: extractor,
	}
}

func Specs() map[Runtime]Spec {
	return map[Runtime]Spec{
		Docker: NewSpec(dockerIgnoreFlags(), dockerPackageExtractor()),
		NPX:    NewSpec(flagNameSet{"-y": {}}, npxPackageExtractor()),
		UVX:    NewSpec(flagNameSet{"--from": {}}, uvxPackageExtractor()),
		Python: NewSpec(flagNameSet{"-m": {}}, pythonPackageExtractor()),
	}
}

func dockerIgnoreFlags() flagNameSet {
	return flagNameSet{
		"-d":        {},
		"--detach":  {},
		"-i":        {},
		"--name":    {},
		"--network": {},
		"--rm":      {},
		"-v":        {},
		"--volume":  {},
	}
}

func dockerPackageExtractor() pkgExtractor {
	return func(args []string) (string, error) {
		skip := true
		skipNext := false

		flagsWithValues := flagNameSet{
			"-e":        {},
			"--env":     {},
			"-p":        {},
			"--name":    {},
			"--network": {},
			"-v":        {},
			"--volume":  {},
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
				if _, ok := flagsWithValues[arg]; ok {
					skipNext = true
				}
				continue
			}

			return arg, nil
		}

		return "", fmt.Errorf("no %s image found", Docker)
	}
}

func npxPackageExtractor() pkgExtractor {
	return func(args []string) (string, error) {
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			normalizedArg := strings.ToLower(arg)
			if strings.HasPrefix(normalizedArg, "git+") || strings.HasPrefix(normalizedArg, "https://") {
				return "", fmt.Errorf("remote sources are unsupported")
			}
			return arg, nil
		}
		return "", fmt.Errorf("no %s package found", NPX)
	}
}

func uvxPackageExtractor() pkgExtractor {
	return func(args []string) (string, error) {
		for i := 0; i < len(args); i++ {
			arg := strings.TrimSpace(args[i])

			if arg == "--from" && i+1 < len(args) {
				next := strings.ToLower(strings.TrimSpace(args[i+1]))
				if strings.HasPrefix(next, "git+") {
					return "", fmt.Errorf("remote git repositories are unsupported")
				}
				if strings.HasPrefix(next, "https://") {
					return "", fmt.Errorf("arbitrary HTTP repositories are unsupported")
				}

				i++ // Skip the next value
				continue
			}

			if !strings.HasPrefix(arg, "-") {
				return arg, nil
			}
		}

		return "", fmt.Errorf("no %s package found", UVX)
	}
}

func pythonPackageExtractor() pkgExtractor {
	return func(args []string) (string, error) {
		for i, arg := range args {
			if arg == "-m" {
				if i+1 >= len(args) {
					return "", fmt.Errorf("missing module name after -m")
				}
				next := args[i+1]
				if strings.HasPrefix(next, "-") {
					return "", fmt.Errorf("invalid module name after -m: %s", next)
				}
				return next, nil
			}
		}
		return "", fmt.Errorf("no %s module found", Python)
	}
}
