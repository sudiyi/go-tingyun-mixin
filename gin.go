package main

import (
	"fmt"
	"go/ast"
)

func buildGinFramework() *Framework {
	fw := &Framework{}

	fw.controllerImportPackages = [][2]string{
		[2]string{"github.com/TingYunAPM/go/framework/gin", "tingyun_gin"},
	}

	fw.findContextParam = func(funcDecl *ast.FuncDecl) (string, bool) {
		params := funcDecl.Type.Params.List
		for _, param := range params {
			starExpr, ok := param.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			selectorExpr, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			ident, ok := selectorExpr.X.(*ast.Ident)
			if !ok {
				continue
			}
			if "gin" == ident.Name && "Context" == selectorExpr.Sel.Name {
				return param.Names[0].Name, true
			}
		}
		return "", false
	}

	fw.createTingyunActionDefineStmt = func(contextParamName string) ast.Stmt {
		return createStmt(fmt.Sprintf("tyAction := tingyun_gin.FindAction(%s)", contextParamName))
	}

	return fw
}
