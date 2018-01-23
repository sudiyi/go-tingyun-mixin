[![Go Report Card](https://goreportcard.com/badge/github.com/sudiyi/go-tingyun-mixin)](https://goreportcard.com/report/github.com/sudiyi/go-tingyun-mixin)

Name
====

go-tingyun-mixin

一个为 Golang Web MVC 项目添加[听云](https://github.com/TingYunAPM/go)探针的代码批量修改工具

Synopsis
========

```shell
./go-tingyun-mixin <code root path> <root package> <framework>
# eg: ./go-tingyun-mixin "/path/to/src/github.com/somebody/some-project" "github.com/somebody/some-project" gin
# <code root path>: 项目代码的根目录
# <root package>: 项目的根包名
# <framework>: 项目使用的框架，目前支持 gin, beego
```

Description
===========

此工具会递归扫描根目录下所有 `.go` 后缀的文件，除了名为 `vender` 的目录下的所有文件

被扫描的 `.go` 文件可能会被此工具改写，所以请注意在使用前务必使用代码版本控制工具做好备份！

所有添加的代码中用到的包也会在 `import` 中自动添加

### component

此工具会扫描所有被标记为 component 的函数：

```go
package some_service

import (
	...
)

//@tingyun:component
func FunctionInService(i int) int {
	...
	j := SubFunctionInService(i)
	...
}

//@tingyun:component
func SubFunctionInService(i int) int {
	...
}
```

将其改写为：

```go
package some_service

import (
	tingyun "github.com/TingYunAPM/go"
	...
)

//@tingyun:component
func FunctionInService(i int, tyComponent *tingyun.Component) int {
	tyComponentSub := tyComponent.CreateComponent("FunctionInService")
	defer tyComponentSub.Finish()
	...
	j := SubFunctionInService(i, tyComponentSub)
	...
}

//@tingyun:component
func SubFunctionInService(i int, tyComponent *tingyun.Component) int {
	tyComponentSub := tyComponent.CreateComponent("FunctionInService")
	defer tyComponentSub.Finish()
	...
}
```

#### 标记

在函数定义上方添加注释作为标记：

```go
//@tingyun:component
func ...
```

标记的函数不能带有接收器参数，即不能是任何类型的方法，否则不会被识别为 component 函数

可以与其它注释放在一起，但注释与注释之间不能有空行（`/*` 形式的注释中的空行不算注释之间的空行）：

```go
// other comments
//@tingyun:component
// other comments
/* other

comments */
```

#### 修改形参

component 函数的形参会在最后追加一个参数： `tyComponent *tingyun.Component`

因此，如果在其它函数中调用了此 component 函数，那么那些函数也必须标记为 component 函数，这样才能传递 `*tingyun.Component`；一直上溯到 controller 函数，只有 controller 函数不标记为 component 函数

#### 修改函数体

会在函数体的开头添加以下代码：在实参 component 名下定义一个此函数对应的子 component，其名称为函数名；并且 defer finish

```go
tyComponentSub := tyComponent.CreateComponent("FunctionInService")
defer tyComponentSub.Finish()
```

#### 修改实参

component 函数中对其它 component 函数的调用实参也会被改写，在最后追加传入 `tyComponentSub`：

直接的调用表达式会被识别：

```go
SubFunctionInService(...)
```

修改为：

```go
SubFunctionInService(..., tyComponentSub)
```

如果调用存在于赋值表达中，则赋值等号的右边只能有一项，且 `=` 和 `:=` 赋值均会被识别：

```go
j := SubFunctionInService(...)
```

修改为：

```go
j := SubFunctionInService(..., tyComponentSub)
```

如果调用的是其它包中的 component 函数：必须以 <包名>.<函数名> 的形式调用，即必须有且只有一个点号；必须是本项目中的包，即包名以命令行参数 `<root package>` 开头；<包名> 可以使用 import 中的别名：

```go
another_service.FunctionInService(...)
```

修改为：

```go
another_service.FunctionInService(..., tyComponentSub)
```

### controller

首先以 gin 为例，此工具会扫描所有被标记为 controller 的函数：

```go
package some_controller

import (
	"github.com/gin-gonic/gin"
	"github.com/somebody/some-project/services/some_service"
)

//@tingyun:controller
func Get(ginCtx *gin.Context) {
	...
	j := some_service.FunctionInService(1)
	...
}
```

将其改写为：

```go
package some_controller

import (
	"github.com/gin-gonic/gin"
	"github.com/somebody/some-project/services/some_service"
	tingyun "github.com/TingYunAPM/go"
	...
)

//@tingyun:controller
func Get(ginCtx *gin.Context) {
	tyAction := tingyun_gin.FindAction(ginCtx)
	tyComponent := tyAction.CreateComponent("Get")
	defer tyComponent.Finish()
	...
	j := some_service.FunctionInService(1, tyComponent)
	...
}
```

#### 标记

在函数定义上方添加注释作为标记：

```go
//@tingyun:controller
func ...
```

形参表中必须至少带有类型为 `*gin.Context` 的参数，否则不会被识别为 controller 函数

同样，可以与其它注释放在一起，但注释与注释之间不能有空行

#### 修改函数体

会在函数体的开头添加以下代码，从 `*gin.Context` 查找 action，`*gin.Context` 的变量名和函数定义中的保持一致；在 action 名下定义一个此函数对应的 component，其名称为函数名；并且 defer finish

```go
tyAction := tingyun_gin.FindAction(ginCtx)
tyComponent := tyAction.CreateComponent("Get")
defer tyComponent.Finish()
```

#### 修改实参

controller 函数中对 component 函数的调用实参也会被改写，在最后追加传入 `tyComponent`，规则与 component 相同

#### beego

与 gin 的处理不同的是，对 controller 函数的要求为，必须包含一个接收器参数：

```go
func (this *SomeController) Get() {
```

action 的定义不同，其中的 `this` 与接收器参数名保持一致：

```go
tyAction := tingyun_beego.FindAction(this.Ctx)
```

