package main

import (
	"fmt"
	"go/ast"
)

func prependComponetParamInDecl(funcDecl *ast.FuncDecl) {
	field := &ast.Field{
		Names: []*ast.Ident{
			&ast.Ident{
				Name: "tyComponent",
			},
		},
		Type: &ast.BasicLit{Value: "*tingyun.Component"},
	}

	params := funcDecl.Type.Params
	paramList := make([]*ast.Field, len(params.List)+1)
	paramList[0] = field
	if len(params.List) > 0 {
		copy(paramList[1:], params.List)
	}
	params.List = paramList
}

func recognizeComponentFunc(solver *Solver, f *File, importPath string) {
	modified := false
	for _, decl := range f.ast.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !checkNotation(f, funcDecl, "tingyun:component") || nil != funcDecl.Recv {
			continue
		}

		solver.componentFuncs.add(importPath, funcDecl.Name.Name)
		prependComponetParamInDecl(funcDecl)
		modified = true
		fmt.Printf("adding tingyun component code in func %s in file %s\n", funcDecl.Name.Name, f.path)
	}

	if modified {
		f.modified = true
		importPackages(f, [][2]string{
			[2]string{"github.com/TingYunAPM/go", "tingyun"},
		})
	}
}

func processComponentCallInComponent(solver *Solver, funcDecl *ast.FuncDecl, importPath string, f *File) {
	stmts := funcDecl.Body.List
	stmtsNew := []ast.Stmt{}
	subDeclaired := false
	for _, stmt := range stmts {
		callExpr, funcName := checkComponentCall(solver, stmt, importPath, f)
		if nil == callExpr {
			stmtsNew = append(stmtsNew, stmt)
			continue
		}

		if checkNotation(f, stmt, "tingyun:subcomponent") {
			if !subDeclaired {
				subDeclaired = true
				stmtsNew = append(stmtsNew, createStmt("var tyComponentSub *tingyun.Component"))
			}
			stmtsNew = append(stmtsNew, createStmt(fmt.Sprintf(`tyComponentSub = tyComponent.CreateComponent("%s")`, funcName)))
			prependComponetParamInCall(callExpr, "tyComponentSub")
			stmtsNew = append(stmtsNew, stmt)
			stmtsNew = append(stmtsNew, createStmt("tyComponentSub.Finish()"))
		} else {
			prependComponetParamInCall(callExpr, "tyComponent")
			stmtsNew = append(stmtsNew, stmt)
		}
	}
	funcDecl.Body.List = stmtsNew
}

func processComponentFunc(solver *Solver, f *File, importPath string) {
	for _, decl := range f.ast.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !checkNotation(f, funcDecl, "tingyun:component") || nil != funcDecl.Recv {
			continue
		}

		processComponentCallInComponent(solver, funcDecl, importPath, f)
	}
}
