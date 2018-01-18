package main

import (
	"fmt"
	"go/ast"
)

func processComponentCallInController(solver *Solver, funcDecl *ast.FuncDecl, importPath, contextParamName string, f *File) bool {
	stmts := funcDecl.Body.List
	stmtsNew := []ast.Stmt{}
	modified := false
	for _, stmt := range stmts {
		callExpr, funcName := checkComponentCall(solver, stmt, importPath, f)
		if nil == callExpr {
			stmtsNew = append(stmtsNew, stmt)
			continue
		}

		if !modified {
			modified = true
			stmtsNew = append(stmtsNew, solver.framework.createTingyunActionDefineStmt(contextParamName))
			stmtsNew = append(stmtsNew, createStmt("var tyComponent *tingyun.Component"))
		}

		stmtsNew = append(stmtsNew, createStmt(fmt.Sprintf(`tyComponent = tyAction.CreateComponent("%s")`, funcName)))
		prependComponetParamInCall(callExpr, "tyComponent")
		stmtsNew = append(stmtsNew, stmt)
		stmtsNew = append(stmtsNew, createStmt("tyComponents.Finish()"))
	}
	funcDecl.Body.List = stmtsNew
	return modified
}

func processControllerFunc(solver *Solver, f *File, importPath string) {
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

		if !processComponentCallInController(solver, funcDecl, importPath, contextParamName, f) {
			continue
		}
		modified = true
		fmt.Printf("adding tingyun controller code in func %s in file %s\n", funcDecl.Name.Name, f.path)
	}

	if modified {
		f.modified = true
		importPackages(f, solver.framework.controllerImportPackages)
	}
}
