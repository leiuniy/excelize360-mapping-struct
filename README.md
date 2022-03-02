# excelize360-mapping-struct

golang 基于[excelize360](https://xuri.me/excelize/zh-hans/utils.html#SetPanes)实现的excel解析工具,通过结构体字段tag实现映射数据到结构体，并加入了一些预期的错误格式，可根据实际运用情况加入更多错误模板，具体使用方式clone本仓库代码，可直接在本地运行调试。

### 支持的tag标签

| tag属性     | 说明                                              |
|---------|-------------------------------------------------|
| mapping | 将数据进行转义，一般数据表格中都是以明文输入输出数据，但在实际代码业务中只用0或1表示状态的场景。例：`mapping(已完成:1,未完成:2)` |
| unique  | 标识唯一性，在同列中唯一。例：`unique(true)`                                                |
| date    | 时间格式化，该属性分两部分，第一个参数：该单元格的时间日期格式（注意该格式非在数据表格中明文显示的格式，要查看该单元格在excel中的时间格式设置）；第二个参数：将转化到的格式。例：`date(01-02-06,2006-01-02)`                                             |
| name    | 指定该属性所在字段绑定的excel列名称，一般情况下数据表格都会指定每一列代表的含义，该属性十分重要，也是映射数据的主要参考依据，需要保持excel列名跟name属性保持一致，例：`name(*名称)`                                                |

以上tag属性统一作用在 `excel` 标签下，如某个字段需要绑定多个属性，属性间用 `“;”` 分隔。
结构体定义示例：

```
type Project struct {
	Name       string      `json:"project_name" excel:"name(*名称);unique(true)"`
	StartTime  *string     `json:"start_time" excel:"name(开始时间);date(01-02-06,2006-01-02)"`
	Status     int         `json:"Status" excel:"name(状态);mapping(已完成:1,未完成:2)"`
	Statistics *Statistics `json:"statistics"`
	Result     struct {
		Count *int `json:"count" excel:"name(总数)"`
	} `json:"result"`
	Time time.Time `json:"time"`
}

type Statistics struct {
	Num int `json:"num" excel:"name(数量)"`
}
```
### 自定义验证接口
工具内置一个接口，可根据需求选择性实现该接口。 

```
type MappingStruct interface {
	ExcelRowProcess(res *Result) error
}
```
通过已经绑定execl标签的结构体实现接口，可对每一行数据（其实就是可以把该结构体看作是每一行数据的模板，数据进行映射操作时每一行映射完都会调用一次该接口）进行自定义校验或进行数据改值（每一行数据都会自动创建新的指针指向并复制初始指针传入的内容）等操作，说明：如果在实现的接口中需要事务性的操作，可以将相关存储引擎通过初始指针进行传递。

接口实现示例：

```
func (p *Project) ExcelRowProcess(res *excel.Result) error {
	log.Println(p.Statistics.Num, *p.Result.Count)
	valid := validation.Validation{}
	valid.Match(p.Name, regexp.MustCompile("^[-_a-zA-Z一-龥]{1,64}$"), strconv.Itoa(excel.ParamUnqualified))
	for _, err := range valid.Errors {
		errType, _ := strconv.Atoi(err.Key)
		res.AddError(errType, "名称")
	}
	return nil
}
```

```
p := new(Project)
p.Time = time.Now() //每一行数据都可复用
processor, err := excel.NewProcessor(p, true)
if err != nil {
    log.Println(err.Error())
    return
}
```
`NewProcessor`： new一个处理器，第二个参数表示是否开启验证，为false的话，自定义的验证就不会被调用了。
接口提供返回error参数，注意如果返回参数不为空，解析会立马停止，并返回该错误，一般在致命错误下返回该错误，一般的验证错误可见下边数据行验证错误的说明。
### Result——内置的错误处理工具与数据映射结果处理

result提供了四个方法，可在实现自定义验证接口中与映射解析返回的结果参数获取该对象进行方法调取。
| 方法名称 | 说明 |
|------|----|
|   AddError   |  主要用于自定义验证接口中，可结合下边介绍错误处理模板添加行错误，提供了一个错误码（错误格式的编码）参数与错误格式化值参数，类似golang中`fmt.Sprintf` 方法 |
|   HasError   |  用于数据解析后，判断excel解析结果是否存在错误，返回错误列表，错误列表由行号+错误信息集合组成，标识了哪一行数据产生了哪些错误  |
|   List   |  返回当前已经完成解析数据行的集合，注意返回类型是一个`[]interface{}`  |
|   Format   |  将已经完成解析数据行的集合通过json序列化反序列化方法在解析到指定的对象中  |


### 数据行验证错误
数据解析过程中会对每一行数据进行解析检验，发现不符合要求的数据单元，并不会立即结束解析，而是会将所有行的错误进行整合，最终将错误结果以 **行号+错误信息集合** 当作一项放入到错误集合中返回最终结果。
####内部验证错误
内部验证错误主要与excel格式的错误或预期定义错误息息相关，错误集合介绍如下：
| 格式 | 组成 | 说明 |
|----|----|----|
|  %s[%s]不可重复  |  列名[列值]  |  unique属性，在做列唯一性校验时提示的错误  |
|  %s单元格格式错误  |  列名  |  date属性，excel设置的单元格时间与结构体date属性第一个参数不一致时提示的错误  |
|  %s单元格存在非法输入  |  列名  |  mapping属性，当单元格输入的值不在mapping定义的转义列表中时提示的错误  |
|  %s单元格非法输入,参数非bool类型值  |  列名  |  单元格输入的值与结构体对应字段类型不一致输出的错误  |
|  %s单元格非法输入,参数非整形数值  |  列名  |  单元格输入的值与结构体对应字段类型不一致输出的错误  |
|  %s单元格非法输入,参数非浮点型数值  |  列名  |  单元格输入的值与结构体对应字段类型不一致输出的错误  |

####自定义错误
自定义错误工具也定义了一些可直接使用的模板，也可根据实际需求自定义，在自定义的验证接口中使用，接口已有错误模板如下：
| 错误码 | 格式 | 组成 | 说明 |
|----|----|----|----|
|  ParamCannotBeEmpty  |  %s参数不可为空  |  自定义(例：`res.AddError(excel.ParamCannotBeEmpty, "名称")`)  |  适用参数空值  |
|  ParamUnqualified  |  %s参数格式不正确  |  自定义  |  适用正则校验  |
|  AlreadyExists  |  %s已存在  |  自定义  |  适用判断已存在  |
|  NotExist  |  %s不存在  |  自定义  |  适用判断不存在  |
|  NotInConfigurationItems  |  %s不在配置项中  |  自定义  |  适用输入的数据不在指定的集合中  |
|  TimeFormatError  |  %s时间格式错误  |  自定义  |  适用时间格式错误  |
|  DataOutsideExpectedLimits  |  %s数据不在预期限制范围  |  自定义  |  适用数据在一定范围内的校验  |
|  ParamInvalid  |  %s参数验证失败  |  自定义  |  适用没有具体定义的错误  |

自定义错误格式模板信息，直接在工具文件头部已列出的模板下进行添加即可。

```
const (
	ParamError = iota
	ParamInvalid
	ParamUnqualified
	AlreadyExists
	NotExist
	NotInConfigurationItems
	TimeFormatError
	DataOutsideExpectedLimits
	ParamCannotBeEmpty
)

var errFormatList = map[int]excelErr{
	ParamCannotBeEmpty:        {"%s参数不可为空"},
	ParamUnqualified:          {"%s参数格式不正确"},
	AlreadyExists:             {"%s已存在"},
	NotExist:                  {"%s不存在"},
	NotInConfigurationItems:   {"%s不在配置项中"},
	TimeFormatError:           {"%s时间格式错误"},
	DataOutsideExpectedLimits: {"%s数据不在预期限制范围"},
	ParamInvalid:              {"%s参数验证失败"},
}
```

其他具体实现以及更多详细使用方式 `main` 方法中已经实现












