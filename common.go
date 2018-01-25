package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func createStmt(s string) ast.Stmt {
	return &ast.ExprStmt{X: &ast.BasicLit{Value: s}}
}

func parseFile(solver *Solver, fpath string) *File {
	fileSet := token.NewFileSet()
	fileAst, err := parser.ParseFile(fileSet, fpath, nil, parser.ParseComments)
	if nil != err {
		panic(err)
	}
	fileCmtMap := ast.NewCommentMap(fileSet, fileAst, fileAst.Comments)

	f := &File{
		ast:        fileAst,
		fileSet:    fileSet,
		commentMap: fileCmtMap,
		path:       fpath,
		imports:    map[string]string{},
	}
	for _, im := range f.ast.Imports {
		path := im.Path.Value[1 : len(im.Path.Value)-1]
		if !strings.HasPrefix(path, solver.basePackagePath) {
			continue
		}
		path = path[len(solver.basePackagePath):]
		if nil != im.Name {
			f.imports[im.Name.Name] = path
		} else {
			es := strings.Split(path, "/")
			f.imports[es[len(es)-1]] = path
		}
	}
	return f
}

func importPackages(f *File, names [][2]string) {
	for _, pair := range names {
		if len(pair[1]) > 0 {
			astutil.AddNamedImport(f.fileSet, f.ast, pair[1], pair[0])
		} else {
			astutil.AddImport(f.fileSet, f.ast, pair[0])
		}
	}
	ast.SortImports(f.fileSet, f.ast)
	f.modified = true
}

func checkNotation(f *File, node ast.Node, notation string) bool {
	cmts := f.commentMap.Filter(node).Comments()
	if 0 == len(cmts) {
		return false
	}
	for _, line := range cmts[0].List {
		if "//@"+notation == line.Text {
			return true
		}
	}
	return false
}

func prependStatements(funcDecl *ast.FuncDecl, prefStmts []ast.Stmt) {
	stmts := make([]ast.Stmt, len(prefStmts)+len(funcDecl.Body.List))
	copy(stmts, prefStmts)
	if len(funcDecl.Body.List) > 0 {
		copy(stmts[len(prefStmts):], funcDecl.Body.List)
	}
	funcDecl.Body.List = stmts
}

func checkComponentCall(solver *Solver, expr ast.Expr) *ast.CallExpr {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch funExpr := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		x, ok := funExpr.X.(*ast.Ident)
		if !ok {
			return nil
		}
		packagePath, ok := solver.file.imports[x.Name]
		if !ok || !solver.componentFuncs.check(packagePath, funExpr.Sel.Name) {
			return nil
		}
	case *ast.Ident:
		if !solver.componentFuncs.check(solver.packagePath, funExpr.Name) {
			return nil
		}
	default:
		return nil
	}
	return callExpr
}

func processComponentCall(solver *Solver, stmts []ast.Stmt, componentVarName string) {
	for _, stmt := range stmts {
		var callExpr *ast.CallExpr
		switch s := stmt.(type) {
		case *ast.IfStmt:
			processComponentCall(solver, s.Body.List, componentVarName)
		case *ast.ForStmt:
			processComponentCall(solver, s.Body.List, componentVarName)
		case *ast.BlockStmt:
			processComponentCall(solver, s.List, componentVarName)
		case *ast.SwitchStmt:
			processComponentCall(solver, s.Body.List, componentVarName)
		case *ast.CaseClause:
			processComponentCall(solver, s.Body, componentVarName)
		case *ast.AssignStmt:
			if 1 == len(s.Rhs) {
				callExpr = checkComponentCall(solver, s.Rhs[0])
			}
		case *ast.ExprStmt:
			callExpr = checkComponentCall(solver, s.X)
		}

		if nil != callExpr {
			callExpr.Args = append(callExpr.Args, &ast.BasicLit{Value: componentVarName})
		}
	}
}
