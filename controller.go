package main

import (
	"fmt"
	"go/ast"
)

func processControllerFunc(solver *Solver) {
	f := solver.file
	modified := false
	for _, decl := range f.ast.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !checkNotation(f, funcDecl, "tingyun:controller") {
			continue
		}
		contextParamName, ok := solver.framework.findContextParam(funcDecl)
		if !ok {
			continue
		}
		modified = true
		fmt.Printf("adding tingyun controller code in func %s in file %s\n", funcDecl.Name.Name, f.path)

		prependStatements(funcDecl, []ast.Stmt{
			solver.framework.createTingyunActionDefineStmt(contextParamName),
			createStmt(fmt.Sprintf(`tyComponent := tyAction.CreateComponent("%s")`, funcDecl.Name.Name)),
			createStmt("defer tyComponent.Finish()"),
		})

		processComponentCall(solver, funcDecl.Body.List, "tyComponent")
	}

	if modified {
		importPackages(f, solver.framework.controllerImportPackages)
	}
}
