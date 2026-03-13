package contract

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"golang.org/x/tools/go/packages"
)

// BuildContracts scans a Go module rooted at dir and extracts workflow
// contracts from all flicker.Define call sites found in the module.
func BuildContracts(ctx context.Context, dir string) ([]Contract, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir:     dir,
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	// Check for package-level errors.
	var loadErrors []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			loadErrors = append(loadErrors, e.Error())
		}
	}
	if len(loadErrors) > 0 {
		return nil, fmt.Errorf("package errors: %v", loadErrors)
	}

	var contracts []Contract

	for _, pkg := range pkgs {
		defs := detectDefines(pkg)
		for _, def := range defs {
			c := buildContract(pkg, def)
			contracts = append(contracts, c)
		}
	}

	sort.Slice(contracts, func(i, j int) bool {
		ki := contracts[i].Name + ":" + contracts[i].Version
		kj := contracts[j].Name + ":" + contracts[j].Version
		return ki < kj
	})

	return contracts, nil
}

// defineCall holds the extracted information from a flicker.Define call site.
type defineCall struct {
	name       string
	version    string
	reqType    types.Type
	respType   types.Type
	factoryArg ast.Expr
	pos        token.Pos
}

// buildContract converts a detected Define call into a Contract.
func buildContract(pkg *packages.Package, def defineCall) Contract {
	c := Contract{
		Name:         def.name,
		Version:      def.version,
		RequestType:  resolveTypeShape(def.reqType),
		ResponseType: resolveTypeShape(def.respType),
	}

	// Resolve factory → concrete struct type → Execute method.
	structType, structObj, err := resolveFactory(pkg, def.factoryArg)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return c
	}

	executeBody, err := findExecuteMethod(pkg, structObj)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return c
	}

	// Walk the factory body for providers.
	factoryProviders := extractFactoryProviders(pkg, def.factoryArg)
	c.Providers = append(c.Providers, factoryProviders...)

	// Walk Execute body for steps and providers.
	steps, providers, walkErrors := walkExecuteBody(pkg, executeBody, structType)
	c.Steps = steps
	c.Providers = append(c.Providers, providers...)
	c.Errors = append(c.Errors, walkErrors...)

	if len(c.Errors) == 0 {
		c.Errors = nil
	}
	if len(c.Providers) == 0 {
		c.Providers = nil
	}

	return c
}
