package main

import (
	"go/ast"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type File struct {
	path       string
	ast        *ast.File
	fileSet    *token.FileSet
	commentMap ast.CommentMap

	imports  map[string]string // import name => import path
	modified bool
}

type Framework struct {
	controllerImportPackages      [][2]string
	findContextParam              func(*ast.FuncDecl) (string, bool)
	createTingyunActionDefineStmt func(string) ast.Stmt
}

var frameworksBuilder = map[string]func() *Framework{
	"gin":   buildGinFramework,
	"beego": buildBeegoFramework,
}

type Solver struct {
	framework       *Framework
	files           map[string]*File // file path in project =>
	basePackagePath string
	componentFuncs  ComponentFuncs // import path => func name => true
}

type ComponentFuncs map[string]map[string]bool

func (componentFuncs ComponentFuncs) add(importPath, funcName string) {
	m, ok := componentFuncs[importPath]
	if !ok {
		m = map[string]bool{}
		componentFuncs[importPath] = m
	}
	m[funcName] = true
}

func (componentFuncs ComponentFuncs) check(importPath, funcName string) bool {
	m, ok := componentFuncs[importPath]
	if !ok {
		return false
	}
	return m[funcName]
}

func scanDir(solver *Solver, dirname, pathInProj string, process func(*Solver, *File, string)) {
	files, err := ioutil.ReadDir(dirname)
	if nil != err {
		panic(err)
	}
	for _, file := range files {
		fname := file.Name()
		if file.IsDir() {
			if "vendor" != fname {
				scanDir(solver, path.Join(dirname, fname), path.Join(pathInProj, fname), process)
			}
		} else if strings.HasSuffix(fname, ".go") {
			fpath := path.Join(dirname, fname)
			f, ok := solver.files[fpath]
			if !ok {
				f = parseFile(solver, fpath)
				solver.files[fpath] = f
			}
			process(solver, f, pathInProj)
		}
	}
}

func main() {
	if len(os.Args) < 4 {
		println("usage: ./go-tingyun-mixin <code root path> <root package> <framework(gin|beego...)>")
		return
	}
	baseDir := os.Args[1]
	frameworkBuilder, ok := frameworksBuilder[os.Args[3]]
	if !ok {
		println("invalid framework")
		return
	}

	solver := &Solver{
		framework:       frameworkBuilder(),
		files:           map[string]*File{},
		basePackagePath: os.Args[2],
		componentFuncs:  ComponentFuncs{},
	}
	if solver.basePackagePath[len(solver.basePackagePath)-1] != '/' {
		solver.basePackagePath += "/"
	}

	scanDir(solver, baseDir, "", recognizeComponentFunc)
	scanDir(solver, baseDir, "", processControllerFunc)
	scanDir(solver, baseDir, "", processComponentFunc)

	for _, f := range solver.files {
		if f.modified {
			/*
				fmt.Printf("writing file %s\n", f.path)
				fout, err := os.OpenFile(f.path, os.O_WRONLY, 0664)
				if nil != err {
					fmt.Printf("failed to open file %s: %s\n", f.path, err.Error())
					break
				}
				if err = printer.Fprint(fout, f.fileSet, f.ast); nil != err {
					fmt.Printf("failed to write file %s: %s\n", f.path, err.Error())
					break
				}
				fout.Sync()
				fout.Close()
			*/
			printer.Fprint(os.Stdout, f.fileSet, f.ast)
		}
	}
}
