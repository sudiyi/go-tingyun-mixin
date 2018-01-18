package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
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
		path:       fpath,
		ast:        fileAst,
		fileSet:    fileSet,
		commentMap: fileCmtMap,
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
	importSpecs := make([]*ast.ImportSpec, len(names))
	for i := 0; i < len(names); i++ {
		pair := names[i]
		importSpec := &ast.ImportSpec{}
		importSpec.Path = &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"` + pair[0] + `"`,
		}
		if len(pair[1]) > 0 {
			importSpec.Name = &ast.Ident{
				Name: pair[1],
			}
		}
		importSpecs[i] = importSpec
	}

	for _, decl := range f.ast.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || token.IMPORT != genDecl.Tok {
			continue
		}
		for _, importSpec := range importSpecs {
			genDecl.Specs = append(genDecl.Specs, importSpec)
		}
	}
	ast.SortImports(f.fileSet, f.ast)
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

func checkComponentCallInner(solver *Solver, expr ast.Expr, importPath string, f *File) (*ast.CallExpr, string) {
	var funcName string
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, ""
	}

	switch funExpr := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		x, ok := funExpr.X.(*ast.Ident)
		if !ok {
			return nil, ""
		}
		importPath, ok = f.imports[x.Name]
		if !ok {
			return nil, ""
		}
		if !ok || !solver.componentFuncs.check(importPath, funExpr.Sel.Name) {
			return nil, ""
		}
		funcName = x.Name + "." + funExpr.Sel.Name
	case *ast.Ident:
		if !solver.componentFuncs.check(importPath, funExpr.Name) {
			return nil, ""
		}
		funcName = funExpr.Name
	default:
		return nil, ""
	}
	return callExpr, funcName
}

func checkComponentCall(solver *Solver, stmt ast.Stmt, importPath string, f *File) (*ast.CallExpr, string) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if 1 != len(s.Rhs) {
			return nil, ""
		}
		return checkComponentCallInner(solver, s.Rhs[0], importPath, f)
	case *ast.ExprStmt:
		return checkComponentCallInner(solver, s.X, importPath, f)
	}
	return nil, ""
}

func prependComponetParamInCall(callExpr *ast.CallExpr, componentVarName string) {
	args := make([]ast.Expr, len(callExpr.Args)+1)
	args[0] = &ast.BasicLit{Value: componentVarName}
	if len(callExpr.Args) > 0 {
		copy(args[1:], callExpr.Args)
	}
	callExpr.Args = args
}
