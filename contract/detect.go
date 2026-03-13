package contract

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

const flickerPkgPath = "github.com/flamingoosesoftwareinc/flicker"

// detectDefines finds all flicker.Define[R, Resp](name, version, factory)
// call sites in a package.
func detectDefines(pkg *packages.Package) []defineCall {
	var calls []defineCall

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if !isFlickerFunc(pkg, call, "Define") {
				return true
			}

			if len(call.Args) < 3 {
				return true
			}

			// Extract name and version from first two args.
			name := resolveStringArg(pkg, call.Args[0])
			version := resolveStringArg(pkg, call.Args[1])

			// Extract R and Resp type params from the instantiation.
			reqType, respType := extractDefineTypeParams(pkg, call)

			calls = append(calls, defineCall{
				name:       name,
				version:    version,
				reqType:    reqType,
				respType:   respType,
				factoryArg: call.Args[2],
				pos:        call.Pos(),
			})

			return true
		})
	}

	return calls
}

// isFlickerFunc checks if a call expression is a call to flicker.<funcName>.
func isFlickerFunc(pkg *packages.Package, call *ast.CallExpr, funcName string) bool {
	// Handle both flicker.Define and flicker.Define[R, Resp] (IndexListExpr).
	var fun ast.Expr
	switch f := call.Fun.(type) {
	case *ast.IndexListExpr:
		fun = f.X
	case *ast.IndexExpr:
		fun = f.X
	default:
		fun = call.Fun
	}

	sel, ok := fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != funcName {
		return false
	}

	// Check if the selector target is the flicker package.
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	obj := pkg.TypesInfo.Uses[ident]
	if obj == nil {
		return false
	}

	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return false
	}

	return pkgName.Imported().Path() == flickerPkgPath
}

// resolveStringArg extracts a string value from an AST expression.
// Handles string literals and constants.
func resolveStringArg(pkg *packages.Package, expr ast.Expr) string {
	// Try constant value first.
	tv, ok := pkg.TypesInfo.Types[expr]
	if ok && tv.Value != nil {
		s := tv.Value.ExactString()
		return unquote(s)
	}

	// Try basic literal.
	if lit, ok := expr.(*ast.BasicLit); ok {
		return unquote(lit.Value)
	}

	return "<unresolved>"
}

// unquote strips surrounding double quotes if present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// extractDefineTypeParams extracts the R and Resp type parameters from a
// flicker.Define[R, Resp](...) call expression using type info.
func extractDefineTypeParams(pkg *packages.Package, call *ast.CallExpr) (types.Type, types.Type) {
	// Get the type of the call result — it's *WorkflowDef[R, Resp].
	tv, ok := pkg.TypesInfo.Types[call]
	if !ok {
		return nil, nil
	}

	// Unwrap pointer.
	ptr, ok := tv.Type.(*types.Pointer)
	if !ok {
		return nil, nil
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return nil, nil
	}

	targs := named.TypeArgs()
	if targs == nil || targs.Len() < 2 {
		return nil, nil
	}

	return targs.At(0), targs.At(1)
}

// resolveFactory traces the factory argument (3rd arg of Define) to the
// concrete struct type that implements Workflow[R, Resp].
// It expects a function literal that returns &SomeStruct{...}.
func resolveFactory(
	pkg *packages.Package,
	factoryArg ast.Expr,
) (*types.Struct, types.Object, error) {
	funcLit, ok := factoryArg.(*ast.FuncLit)
	if !ok {
		return nil, nil, fmt.Errorf(
			"factory argument is not a function literal (got %T)",
			factoryArg,
		)
	}

	if funcLit.Body == nil || len(funcLit.Body.List) == 0 {
		return nil, nil, fmt.Errorf("factory function has empty body")
	}

	// Find the return statement.
	var retExpr ast.Expr
	for _, stmt := range funcLit.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if ok && len(ret.Results) > 0 {
			retExpr = ret.Results[0]
			break
		}
	}

	if retExpr == nil {
		return nil, nil, fmt.Errorf("factory function has no return statement")
	}

	// Expect &SomeStruct{...} — unary & with composite lit.
	unary, ok := retExpr.(*ast.UnaryExpr)
	if !ok {
		return nil, nil, fmt.Errorf("factory return is not &SomeStruct{} (got %T)", retExpr)
	}

	compLit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return nil, nil, fmt.Errorf("factory return is not &SomeStruct{} (got &%T)", unary.X)
	}

	// Get the type of the composite literal.
	tv, ok := pkg.TypesInfo.Types[compLit]
	if !ok {
		return nil, nil, fmt.Errorf("could not resolve composite literal type")
	}

	named, ok := tv.Type.(*types.Named)
	if !ok {
		return nil, nil, fmt.Errorf("composite literal type is not named (got %s)", tv.Type)
	}

	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, fmt.Errorf(
			"composite literal type is not a struct (got %s)",
			named.Underlying(),
		)
	}

	return structType, named.Obj(), nil
}

// findExecuteMethod looks up the Execute method on the given struct type
// and returns its AST body.
func findExecuteMethod(pkg *packages.Package, structObj types.Object) (*ast.BlockStmt, error) {
	structName := structObj.Name()

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Name.Name != "Execute" {
				continue
			}

			if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
				continue
			}

			// Check if receiver is *structName.
			recvType := funcDecl.Recv.List[0].Type
			star, ok := recvType.(*ast.StarExpr)
			if !ok {
				continue
			}

			ident, ok := star.X.(*ast.Ident)
			if !ok {
				continue
			}

			if ident.Name == structName {
				return funcDecl.Body, nil
			}
		}
	}

	return nil, fmt.Errorf("execute method not found for type %s", structName)
}

// extractFactoryProviders finds NewProvider calls in the factory function body.
func extractFactoryProviders(pkg *packages.Package, factoryArg ast.Expr) []Provider {
	funcLit, ok := factoryArg.(*ast.FuncLit)
	if !ok {
		return nil
	}

	var providers []Provider

	ast.Inspect(funcLit, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if !isFlickerFunc(pkg, call, "NewProvider") {
			return true
		}

		p, ok := extractProvider(pkg, call)
		if ok {
			providers = append(providers, p)
		}

		return true
	})

	return providers
}

// extractProvider extracts a Provider from a flicker.NewProvider[T](wc, prefix, fn) call.
func extractProvider(pkg *packages.Package, call *ast.CallExpr) (Provider, bool) {
	if len(call.Args) < 2 {
		return Provider{}, false
	}

	prefix := resolveStringArg(pkg, call.Args[1])

	// Get T from the call's type — it's *Provider[T].
	tv, ok := pkg.TypesInfo.Types[call]
	if !ok {
		return Provider{}, false
	}

	ptr, ok := tv.Type.(*types.Pointer)
	if !ok {
		return Provider{}, false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return Provider{}, false
	}

	targs := named.TypeArgs()
	if targs == nil || targs.Len() < 1 {
		return Provider{}, false
	}

	return Provider{
		Prefix: prefix,
		Type:   resolveTypeShape(targs.At(0)),
	}, true
}
