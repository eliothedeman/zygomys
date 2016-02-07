package zygo

import (
	"bytes"
	"errors"
	"fmt"
)

// SexpPackage is a collection of functions modeled after Go's packages which allows for the namespacing of functions
type SexpPackage struct {
	funcs map[string]*SexpFunction
	name  string
}

func PackageFromFuncMap(name string, m map[string]GlispUserFunction) *SexpPackage {
	p := NewSexpPackage(name)

	for k, v := range m {
		p.funcs[k] = MakeUserFunction(k, v)
	}

	return p
}

func NewSexpPackage(name string) *SexpPackage {
	return &SexpPackage{
		funcs: make(map[string]*SexpFunction),
		name:  name,
	}
}

// SexpString encodes this value as a string
func (p *SexpPackage) SexpString() string {
	buff := bytes.NewBuffer(nil)
	buff.WriteString("{\n")
	for k, v := range p.funcs {
		buff.WriteString(fmt.Sprintf("\t%s:\t %s\n", k, v.SexpString()))
	}
	buff.WriteString("}\n")

	return buff.String()
}

func (s *SexpPackage) FmtPackageFuncName(name string) string {
	return fmt.Sprintf("%s/%s", s.name, name)
}

// PackageFunction returns a new package with the given name
func PackageFunction(env *Glisp, name string, args []Sexp) (Sexp, error) {
	var p *SexpPackage

	if len(args) < 1 {
		return SexpNull, NewSexpError(errors.New("Must provided name of package"))
	}

	// make sure that we have a string
	pkgName, ok := args[0].(SexpStr)
	if !ok {
		return SexpNull, NewSexpError(errors.New("First argument must be a str"))
	}

	// check to see of the env already has the package loaded
	if p, ok = env.packages[string(pkgName)]; !ok {
		p = NewSexpPackage(string(pkgName))

		// load our new package
		env.packages[string(pkgName)] = p
	}

	return p, nil
}

// ExportFunction registers a function with the package
func ExportFunction(env *Glisp, name string, args []Sexp) (Sexp, error) {
	var p *SexpPackage

	// first arg should be a package
	if len(args) < 1 {
		return SexpNull, NewSexpError(errors.New("Must Provide a pkg as first argument"))
	}

	p, ok := args[0].(*SexpPackage)
	if !ok {
		return SexpNull, NewSexpError(errors.New("Must Provide a pkg as first argument"))
	}

	// second arg should be a SexpFunction
	if len(args) < 2 {
		return SexpNull, NewSexpError(errors.New("Must provide a func as second argument"))
	}

	f, ok := args[1].(*SexpFunction)
	if !ok {
		return SexpNull, NewSexpError(errors.New("Must provide a func as second argument"))
	}

	// add to the package
	p.funcs[f.name] = f

	return SexpNull, nil
}

// ImportFunction imports a package into the global scope of the environment
func ImportFunction(env *Glisp, name string, args []Sexp) (Sexp, error) {
	// looking for package name
	if len(args) < 1 {
		return SexpNull, NewSexpError(errors.New("Must provide a package name as first argument"))
	}

	pkgName, ok := args[0].(SexpStr)
	if !ok {
		return SexpNull, NewSexpError(errors.New("First argument must be a str"))
	}

	// look up the pacakge in the env
	pkg, ok := env.packages[string(pkgName)]
	if !ok {
		return SexpNull, NewSexpError(fmt.Errorf("Unknown package %s", pkgName))
	}

	// load all of the package's functions into the env
	for k, v := range pkg.funcs {
		env.AddGlobal(pkg.FmtPackageFuncName(k), v)
	}

	return pkg, nil
}
