package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

type docMeta struct {
	Deviations []string
}

type structDoc struct {
	Filename   string
	Filepath   string
	Name       string
	Parameters map[string]bool
	Meta       *docMeta
}

var metaRe = regexp.MustCompile(`@meta\s*(\{[\s\S]*\})`)

func toSnakeCase(s string) string {
	// Insert an underscore before all caps that are followed by lowercase letters
	re1 := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	s = re1.ReplaceAllString(s, "${1}_${2}")

	// Handle cases like "HTTPServer" → "http_server"
	re2 := regexp.MustCompile(`([A-Z])([A-Z][a-z])`)
	s = re2.ReplaceAllString(s, "${1}_${2}")

	return strings.ToLower(s)
}

func goFilesInDir(dir string) ([]string, error) {
	var files []string
	return files, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		files = append(files, path)
		return nil
	})
}

func fileNodeToStructDoc(fileNode *ast.File, path string) (*structDoc, error) {
	// We only care about top-level declarations, since that's where our
	// structs are.
	for _, decl := range fileNode.Decls {
		// Structs are generic declarations
		genDecl, ok := decl.(*ast.GenDecl)
		// Exclude anything that's not a generic declaration or a type
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		if genDecl.Doc == nil {
			continue
		}

		m := metaRe.FindStringSubmatch(genDecl.Doc.Text())
		if len(m) < 1 {
			continue
		}

		var meta docMeta
		if err := json.Unmarshal([]byte(m[1]), &meta); err != nil {
			return nil, fmt.Errorf("failed to parse meta in %s: %w", path, err)
		}
		sdoc := structDoc{
			Meta:     &meta,
			Filename: filepath.Base(path),
			Filepath: path,
		}

		for _, spec := range genDecl.Specs {
			// We're looking only for struct type specifications,
			// everything else we can ignore.
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			sdoc.Name = toSnakeCase(typeSpec.Name.Name)

			params := map[string]bool{}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				name := strings.ToLower(field.Names[0].Name)
				if field.Tag != nil {
					tags := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))

					yamlName, ok := tags.Lookup("yaml")
					if ok {
						name = yamlName
					}

					sophonsTag := tags.Get("sophons")
					params[name] = (sophonsTag == "implemented")
				} else {
					params[name] = false
				}
			}

			sdoc.Parameters = params
			return &sdoc, nil
		}
	}
	return nil, nil
}

func extractStructDocs(dir string) ([]structDoc, error) {
	var structs []structDoc
	files, err := goFilesInDir(dir)
	if err != nil {
		return []structDoc{}, err
	}

	fset := token.NewFileSet()

	for _, file := range files {
		fileNode, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return []structDoc{}, err
		}

		sdoc, err := fileNodeToStructDoc(fileNode, file)
		if err != nil {
			return []structDoc{}, err
		}

		if sdoc != nil {
			structs = append(structs, *sdoc)
		}
	}
	return structs, nil
}

var pageTemplate = `# ansible.builtin.{{ .Doc.Name }}

## Implementation

| Source | Parameters | Deviations |
|--------|------------|------------|
| [{{.Doc.Filename}}](../../{{.Doc.Filepath}}) | {{.ParametersEmoji}} | {{.DeviationsEmoji}} |

## Parameters

| Name | Implemented |
|------|-------------|
{{range $name,$implemented := .Doc.Parameters}}| {{$name}} | {{if $implemented}} :white_check_mark: {{else}} :x: {{end}} |
{{end}}
## Deviations

{{if eq (len .Doc.Meta.Deviations) 0}}None.{{else}}{{range .Doc.Meta.Deviations}}* {{.}}
{{end}}{{end}}
`

func main() {
	t := template.Must(template.New("doc").Parse(pageTemplate))

	inDir := os.Args[1]
	outDir := os.Args[2]

	s, err := extractStructDocs(inDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, su := range s {
		pemoji := ":white_check_mark:"
		for _, v := range su.Parameters {
			if !v {
				pemoji = ":x:"
			}
		}

		demoji := ":white_check_mark:"
		if len(su.Meta.Deviations) > 0 {
			demoji = ":x:"
		}

		vars := struct {
			Doc             structDoc
			ParametersEmoji string
			DeviationsEmoji string
		}{
			Doc:             su,
			ParametersEmoji: pemoji,
			DeviationsEmoji: demoji,
		}

		f, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.md", su.Name)))
		if err != nil {
			log.Fatal(err)
		}
		if err := t.Execute(f, vars); err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
}
