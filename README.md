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
工具内置一个接口，可根据需求选择性实现该接口：

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


















