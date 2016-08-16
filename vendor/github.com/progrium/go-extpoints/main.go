package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gedex/inflector"
)

func processFile(inputPath string) (string, []string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Could not parse file: %s", err)
	}

	packageName := identifyPackage(f)
	if packageName == "" {
		log.Fatalf("Could not determine package name of %s", inputPath)
	}

	var ifaces []string
	for _, decl := range f.Decls {
		if typeName, ok := identifyInterface(decl); ok {
			ifaces = append(ifaces, typeName)
			continue
		}
		if typeName, ok := identifyFuncType(decl); ok {
			ifaces = append(ifaces, typeName)
			continue
		}
	}

	return packageName, ifaces
}

func identifyPackage(f *ast.File) string {
	if f.Name == nil {
		return ""
	}
	return f.Name.Name
}

func identifyFuncType(decl ast.Decl) (typeName string, match bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range genDecl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			if _, ok := typeSpec.Type.(*ast.FuncType); ok {
				if typeSpec.Name != nil {
					typeName = typeSpec.Name.Name
					break
				}
			}
		}
	}
	if typeName == "" {
		return
	}
	match = true
	return
}

func identifyInterface(decl ast.Decl) (typeName string, match bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, spec := range genDecl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				if typeSpec.Name != nil {
					typeName = typeSpec.Name.Name
					break
				}
			}
		}
	}
	if typeName == "" {
		return
	}
	match = true
	return
}

type extensionPoint struct {
	Name string
}

func (i *extensionPoint) Var() string {
	return inflector.Pluralize(i.Name)
}

func (i *extensionPoint) Type() string {
	return strings.ToLower(i.Name[0:1]) + i.Name[1:] + "Ext"
}

func extensionPoints(ifaces []string) []extensionPoint {
	var extpoints []extensionPoint
	for _, iface := range ifaces {
		extpoints = append(extpoints, extensionPoint{iface})
	}
	return extpoints
}

type templateData struct {
	Package         string
	ExtensionPoints []extensionPoint
}

func renderExtpoints(path, packageName string, ifaces []string) error {
	output, err := os.Create(path)
	if err != nil {
		log.Fatalf("Could not open output file: %s", err)
	}
	defer output.Close()
	outputTemplate := template.Must(template.New("render").Parse(extpointsTemplate))
	return outputTemplate.Execute(output, templateData{packageName, extensionPoints(ifaces)})
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("extpoints: ")

	packagePath := "./extpoints"
	if len(os.Args) > 1 {
		packagePath = os.Args[1]
	}
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		log.Fatal("Unable to find package for extpoints, not found:", packagePath)
	}

	var packageName string
	var ifaces, ifacesAll []string
	ifacesAllowed := make(map[string]struct{})

	if len(os.Args) > 2 {
		for _, iface := range os.Args[2:] {
			ifacesAllowed[iface] = struct{}{}
		}
	}

	files, _ := ioutil.ReadDir(packagePath)
	for _, file := range files {
		if file.Name() != "extpoints.go" && !strings.HasSuffix(file.Name(), "_ext.go") {
			path := filepath.Join(packagePath, file.Name())
			log.Printf("Processing file %s", path)
			packageName, ifaces = processFile(path)
			if len(ifacesAllowed) > 0 {
				var ifacesFiltered []string
				for _, iface := range ifaces {
					_, allowed := ifacesAllowed[iface]
					if allowed {
						ifacesFiltered = append(ifacesFiltered, iface)
					}
				}
				ifaces = ifacesFiltered
			}
			log.Printf("Found interfaces: %#v", ifaces)
			ifacesAll = append(ifacesAll, ifaces...)
		}
	}

	path := filepath.Join(packagePath, "extpoints.go")
	log.Printf("Writing file %s", path)
	err := renderExtpoints(path, packageName, ifacesAll)
	if err != nil {
		log.Fatalf("Could not write extpoints.go file: %s", err)
	}
}
