package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// eachImport calls visit for every import of every non-test .go file under root.
// file is relative to root with forward slashes, e.g. "article/usecase/create.go".
func eachImport(t *testing.T, root string, visit func(file, importPath string)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		for _, spec := range parsed.Imports {
			visit(filepath.ToSlash(relative), strings.Trim(spec.Path.Value, `"`))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

// Imports in modules/ and platform/ follow the architecture rules: modules reach each other only through
// contract/, and only from transport/ or gateway/; domain/ and contract/ use the standard library only;
// usecase/ uses only the standard library and its own domain/ and contract/; transport/ and gateway/ do not
// import each other; platform/ knows no module.
func TestImportRules(t *testing.T) {
	const modulePath = "github.com/snykk/go-rest-boilerplate"
	const modules = modulePath + "/internal/modules/"
	stdlib := func(importPath string) bool {
		first, _, _ := strings.Cut(importPath, "/")
		return !strings.Contains(first, ".")
	}
	// layer is the folder right below a module: "usecase" for "article/usecase/create.go", "" for files at
	// the module root such as "article/module.go".
	layer := func(file string) string {
		if parts := strings.Split(file, "/"); len(parts) >= 3 {
			return parts[1]
		}
		return ""
	}
	inModules := func(importPath, part string) bool {
		return strings.HasPrefix(importPath, modules) && strings.Contains(importPath, part)
	}

	tCases := []struct {
		name      string
		root      string
		violation func(file, importPath string) string // why the import is not allowed, or ""
	}{
		{
			name: "modules talk only through contract/, only from transport/ or gateway/",
			root: "../modules",
			violation: func(file, importPath string) string {
				rest, ok := strings.CutPrefix(importPath, modules)
				if !ok {
					return ""
				}
				target := strings.SplitN(rest, "/", 2)
				if target[0] == strings.Split(file, "/")[0] {
					return ""
				}
				if len(target) != 2 || target[1] != "contract" {
					return "another module is reachable only through its contract/ package"
				}
				if folder := layer(file); folder != "transport" && folder != "gateway" {
					return "only transport/ and gateway/ may use another module"
				}
				return ""
			},
		},
		{
			name: "domain/ and contract/ use the standard library only",
			root: "../modules",
			violation: func(file, importPath string) string {
				if folder := layer(file); (folder == "domain" || folder == "contract") && !stdlib(importPath) {
					return folder + "/ may use the standard library only"
				}
				return ""
			},
		},
		{
			name: "usecase/ uses only the standard library and its own module's domain/ and contract/",
			root: "../modules",
			violation: func(file, importPath string) string {
				if layer(file) != "usecase" || stdlib(importPath) {
					return ""
				}
				own := modules + strings.Split(file, "/")[0] + "/"
				for _, allowed := range []string{own + "domain", own + "contract"} {
					if importPath == allowed || strings.HasPrefix(importPath, allowed+"/") {
						return ""
					}
				}
				return "usecase/ may use only the standard library and its own domain/ and contract/"
			},
		},
		{
			name: "transport/ and gateway/ do not import each other",
			root: "../modules",
			violation: func(file, importPath string) string {
				switch {
				case layer(file) == "transport" && inModules(importPath, "/gateway/"):
					return "transport/ must not import gateway/"
				case layer(file) == "gateway" && inModules(importPath, "/transport/"):
					return "gateway/ must not import transport/"
				}
				return ""
			},
		},
		{
			name: "platform/ knows no module",
			root: "../platform",
			violation: func(file, importPath string) string {
				if strings.HasPrefix(importPath, modules) || strings.HasPrefix(importPath, modulePath+"/internal/app") {
					return "platform must not depend on modules or app"
				}
				return ""
			},
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			eachImport(t, tc.root, func(file, importPath string) {
				if why := tc.violation(file, importPath); why != "" {
					t.Errorf("%s imports %s: %s", file, importPath, why)
				}
			})
		})
	}
}
