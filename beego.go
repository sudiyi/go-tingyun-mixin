package main

import (
	"fmt"
	"go/ast"
)

func buildBeegoFramework() *Framework {
	fw := &Framework{}

	fw.controllerImportPackages = [][2]string{
		[2]string{"github.com/TingYunAPM/go", "tingyun"},
		[2]string{"github.com/TingYunAPM/go/framework/beego", "tingyun_beego"},
	}

	fw.findContextParam = func(funcDecl *ast.FuncDecl) (string, bool) {
		recvs := funcDecl.Recv.List
		if len(recvs) < 1 {
			return "", false
		}
		return recvs[0].Names[0].Name, true
	}

	fw.createTingyunActionDefineStmt = func(contextParamName string) ast.Stmt {
		return createStmt(fmt.Sprintf("tyAction := tingyun_beego.FindAction(%s.Ctx)", contextParamName))
	}

	return fw
}
