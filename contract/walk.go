package contract

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// walkExecuteBody walks the AST of an Execute method body and extracts
// steps, providers, and errors.
func walkExecuteBody(
	pkg *packages.Package,
	body *ast.BlockStmt,
	structType *types.Struct,
) ([]Step, []Provider, []string) {
	w := &executeWalker{
		pkg:        pkg,
		structType: structType,
	}
	w.walkBlock(body)
	return w.steps, w.providers, w.errors
}

type executeWalker struct {
	pkg        *packages.Package
	structType *types.Struct
	steps      []Step
	providers  []Provider
	errors     []string
}

func (w *executeWalker) walkBlock(block *ast.BlockStmt) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		w.walkStmt(stmt)
	}
}

func (w *executeWalker) walkStmt(stmt ast.Stmt) {
	w.inspectNode(stmt)
}

// inspectNode walks an AST node for framework calls, but stops recursing
// into function literals that belong to recognized framework calls (e.g.
// branch bodies in Parallel, step functions in Run) to avoid double-counting.
func (w *executeWalker) inspectNode(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// If this is a Parallel call, process it (which recurses into
		// branches via sub-walkers) and stop recursing into its children.
		if isFlickerFunc(w.pkg, call, "Parallel") && len(call.Args) >= 2 {
			w.extractParallelStep(call)
			return false
		}

		// For Run/WaitForEvent, process and stop recursing (the fn arg
		// is step implementation, not framework calls to extract).
		if isFlickerFunc(w.pkg, call, "Run") && len(call.Args) >= 3 {
			w.extractRunStep(call)
			return false
		}

		if isFlickerFunc(w.pkg, call, "WaitForEvent") && len(call.Args) >= 3 {
			w.extractWaitForEventStep(call)
			return false
		}

		if isFlickerFunc(w.pkg, call, "NewProvider") {
			p, ok := extractProvider(w.pkg, call)
			if ok {
				w.providers = append(w.providers, p)
			}
			return false
		}

		// Check wc method calls.
		if w.isSleepUntilCall(call) {
			w.steps = append(w.steps, Step{
				Name:     "_sleep.until:*",
				Kind:     StepKindProvider,
				StepType: "sleep_until",
			})
			return true
		}

		if w.isTimeNowCall(call) {
			w.steps = append(w.steps, Step{
				Name:     "_time.now:*",
				Kind:     StepKindProvider,
				StepType: "provider",
			})
			return true
		}

		if w.isIDNewCall(call) {
			w.steps = append(w.steps, Step{
				Name:     "_id.new:*",
				Kind:     StepKindProvider,
				StepType: "provider",
			})
			return true
		}

		return true
	})
}

func (w *executeWalker) extractRunStep(call *ast.CallExpr) {
	nameExpr := call.Args[2]
	name, kind := w.classifyStepName(nameExpr)

	if kind == StepKindDynamic {
		w.errors = append(w.errors, "step name is not a string literal or constant: "+name)
	}

	step := Step{
		Name:     name,
		Kind:     kind,
		StepType: "run",
	}

	// Extract T from call result type.
	if ts := w.extractCallTypeParam(call); ts != nil {
		step.Type = ts
	}

	w.steps = append(w.steps, step)
}

func (w *executeWalker) extractWaitForEventStep(call *ast.CallExpr) {
	nameExpr := call.Args[2]
	name, kind := w.classifyStepName(nameExpr)

	if kind == StepKindDynamic {
		w.errors = append(w.errors, "step name is not a string literal or constant: "+name)
	}

	step := Step{
		Name:     name,
		Kind:     kind,
		StepType: "wait_for_event",
	}

	if ts := w.extractCallTypeParam(call); ts != nil {
		step.Type = ts
	}

	w.steps = append(w.steps, step)
}

func (w *executeWalker) extractParallelStep(call *ast.CallExpr) {
	step := Step{
		Name:     "_parallel",
		Kind:     StepKindProvider,
		StepType: "parallel",
	}

	// Extract branches from remaining args (after ctx, wc).
	for _, arg := range call.Args[2:] {
		branch := w.extractBranch(arg)
		if branch != nil {
			step.Branches = append(step.Branches, *branch)
		}
	}

	w.steps = append(w.steps, step)
}

func (w *executeWalker) extractBranch(expr ast.Expr) *Branch {
	// Expect flicker.NewBranch(name, fn)
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	if !isFlickerFunc(w.pkg, call, "NewBranch") || len(call.Args) < 2 {
		return nil
	}

	name := resolveStringArg(w.pkg, call.Args[0])

	// Get the function literal body.
	funcLit, ok := call.Args[1].(*ast.FuncLit)
	if !ok {
		return nil
	}

	// Create a sub-walker for the branch body.
	subWalker := &executeWalker{
		pkg:        w.pkg,
		structType: w.structType,
	}
	subWalker.walkBlock(funcLit.Body)

	// Propagate errors up.
	w.errors = append(w.errors, subWalker.errors...)
	w.providers = append(w.providers, subWalker.providers...)

	return &Branch{
		Name:  name,
		Steps: subWalker.steps,
	}
}

// classifyStepName determines if a step name is a literal, constant, or dynamic.
func (w *executeWalker) classifyStepName(expr ast.Expr) (string, StepKind) {
	// Check if it's a string literal.
	if lit, ok := expr.(*ast.BasicLit); ok {
		s := lit.Value
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1], StepKindLiteral
		}
		return s, StepKindLiteral
	}

	// Check if it's a constant (via types.Info).
	tv, ok := w.pkg.TypesInfo.Types[expr]
	if ok && tv.Value != nil {
		val := tv.Value.ExactString()
		// Strip quotes from string constants.
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		return val, StepKindConstant
	}

	// Dynamic — cannot resolve at compile time.
	return "<dynamic>", StepKindDynamic
}

// extractCallTypeParam extracts the type parameter T from the result of a
// generic function call like flicker.Run[T](...) → (*T, error).
func (w *executeWalker) extractCallTypeParam(call *ast.CallExpr) *TypeShape {
	tv, ok := w.pkg.TypesInfo.Types[call]
	if !ok {
		return nil
	}

	// Result is (*T, error) — a tuple.
	tuple, ok := tv.Type.(*types.Tuple)
	if !ok {
		return nil
	}

	if tuple.Len() < 1 {
		return nil
	}

	// First element is *T.
	ptr, ok := tuple.At(0).Type().(*types.Pointer)
	if !ok {
		return nil
	}

	ts := resolveTypeShape(ptr.Elem())
	return &ts
}

// isSleepUntilCall checks if a call is wc.SleepUntil(ctx, t).
func (w *executeWalker) isSleepUntilCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "SleepUntil" {
		return false
	}

	return w.isWorkflowContextReceiver(sel.X)
}

// isTimeNowCall checks if a call is wc.Time.Now(ctx).
func (w *executeWalker) isTimeNowCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Now" {
		return false
	}

	// sel.X should be wc.Time
	innerSel, ok := sel.X.(*ast.SelectorExpr)
	if !ok || innerSel.Sel.Name != "Time" {
		return false
	}

	return w.isWorkflowContextReceiver(innerSel.X)
}

// isIDNewCall checks if a call is wc.ID.New(ctx).
func (w *executeWalker) isIDNewCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "New" {
		return false
	}

	innerSel, ok := sel.X.(*ast.SelectorExpr)
	if !ok || innerSel.Sel.Name != "ID" {
		return false
	}

	return w.isWorkflowContextReceiver(innerSel.X)
}

// isWorkflowContextReceiver checks if an expression refers to a field of the
// workflow struct that is a *flicker.WorkflowContext. This handles patterns
// like w.wc where w is the receiver and wc is the WorkflowContext field.
func (w *executeWalker) isWorkflowContextReceiver(expr ast.Expr) bool {
	// Check the type of the expression.
	tv, ok := w.pkg.TypesInfo.Types[expr]
	if !ok {
		return false
	}

	return isWorkflowContextType(tv.Type)
}

// isWorkflowContextType checks if a type is *flicker.WorkflowContext.
func isWorkflowContextType(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}

	return named.Obj().Name() == "WorkflowContext" &&
		named.Obj().Pkg() != nil &&
		named.Obj().Pkg().Path() == flickerPkgPath
}
