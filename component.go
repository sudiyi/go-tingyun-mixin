package main

import (
	"fmt"
	"go/ast"
)

func recognizeComponentFunc(solver *Solver) {
	f := solver.file
	modified := false
	for _, decl := range f.ast.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !checkNotation(f, funcDecl, "tingyun:component") || nil != funcDecl.Recv {
			continue
		}
		modified = true
		fmt.Printf("adding tingyun component code in func %s in file %s\n", funcDecl.Name.Name, f.path)

		solver.componentFuncs.add(solver.packagePath, funcDecl.Name.Name)

		funcDecl.Type.Params.List = append(funcDecl.Type.Params.List, &ast.Field{
			Names: []*ast.Ident{&ast.Ident{Name: "tyComponent"}},
			Type:  &ast.BasicLit{Value: "*tingyun.Component"},
		})

		prependStatements(funcDecl, []ast.Stmt{
			createStmt(fmt.Sprintf(`tyComponentSub := tyComponent.CreateComponent("%s")`, funcDecl.Name.Name)),
			createStmt("defer tyComponentSub.Finish()"),
		})
	}

	if modified {
		importPackages(f, [][2]string{
			[2]string{"github.com/TingYunAPM/go", "tingyun"},
		})
	}
}

func processComponentFunc(solver *Solver) {
	f := solver.file
	for _, decl := range f.ast.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || !checkNotation(f, funcDecl, "tingyun:component") || nil != funcDecl.Recv {
			continue
		}

		processComponentCall(solver, funcDecl.Body.List, "tyComponentSub")
	}
}
