package generator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/hashicorp/go-multierror"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

type templateData struct {
	Package string
	Name    string
	KeyType *goType
	ValType *goType
}

type goType struct {
	Modifiers  string
	ImportPath string
	ImportName string
	Name       string
}

func (t *goType) String() string {
	if t.ImportName != "" {
		return t.Modifiers + t.ImportName + "." + t.Name
	}

	return t.Modifiers + t.Name
}

func (t *goType) IsPtr() bool {
	return strings.HasPrefix(t.Modifiers, "*")
}

func (t *goType) IsSlice() bool {
	return strings.HasPrefix(t.Modifiers, "[]")
}

var partsRe = regexp.MustCompile(`^([\[\]\*]*)(.*?)(\.\w*)?$`)

func parseType(str string) (*goType, error) {
	parts := partsRe.FindStringSubmatch(str)
	if len(parts) != 4 {
		return nil, errors.New("type must be in the form []*github.com/import/path.Name")
	}

	t := &goType{
		Modifiers:  parts[1],
		ImportPath: parts[2],
		Name:       strings.TrimPrefix(parts[3], "."),
	}

	if t.Name == "" {
		t.Name = t.ImportPath
		t.ImportPath = ""
	}

	if t.ImportPath != "" {
		p, err := packages.Load(&packages.Config{Mode: packages.NeedName}, t.ImportPath)
		if err != nil {
			return nil, err
		}
		if len(p) != 1 {
			return nil, fmt.Errorf("package: %s not found", t.ImportPath)
		}

		t.ImportName = p[0].Name
	}

	return t, nil
}

func Generate(name, fileName, keyType, valueType, workingDirectory string) error {
	data, err := getData(name, keyType, valueType, workingDirectory)
	if err != nil {
		return err
	}

	if len(fileName) == 0 {
		fileName = "generated_" + strcase.ToSnake(data.Name) + ".go"
	}

	filePath := filepath.Join(workingDirectory, fileName)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return writeTemplate(filePath, data)
}

func getData(name string, keyType string, valueType string, workingDirectory string) (templateData, error) {
	var data templateData

	var errs error
	genPkg := getPackage(workingDirectory)
	if genPkg == nil {
		errs = fmt.Errorf("unable to find package info for %s", workingDirectory)
	}
	if genPkg.Name == "" {
		err := fmt.Errorf("unable to find package name for %s", workingDirectory)
		if errs == nil {
			errs = err
		} else {
			errs = multierror.Append(errs, err)
		}
	}
	if errs != nil {
		return templateData{}, errs
	}

	var err error
	data.Name = name
	data.Package = genPkg.Name
	data.KeyType, err = parseType(keyType)
	if err != nil {
		return templateData{}, fmt.Errorf("failed to parse key type: %w", err)
	}

	data.ValType, err = parseType(valueType)
	if err != nil {
		return templateData{}, fmt.Errorf("failed to parse value type: %w", err)
	}

	// if we are inside the same package as the type we don't need an import and can refer directly to the type
	if genPkg.PkgPath == data.ValType.ImportPath {
		data.ValType.ImportName = ""
		data.ValType.ImportPath = ""
	}
	if genPkg.PkgPath == data.KeyType.ImportPath {
		data.KeyType.ImportName = ""
		data.KeyType.ImportPath = ""
	}

	return data, nil
}

func getPackage(dir string) *packages.Package {
	p, _ := packages.Load(&packages.Config{
		Dir: dir,
	}, ".")

	if len(p) != 1 {
		return nil
	}

	return p[0]
}

func writeTemplate(filepath string, data templateData) error {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	src, err := imports.Process(filepath, buf.Bytes(), nil)
	if err != nil {
		return fmt.Errorf("unable to gofmt: %w", err)
	}

	if err := ioutil.WriteFile(filepath, src, 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func lcFirst(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
