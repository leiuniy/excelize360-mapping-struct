# excelize360-mapping-struct

golang 基于[excelize360](https://xuri.me/excelize/zh-hans/utils.html#SetPanes)实现的excel解析工具,通过结构体字段tag实现映射数据到结构体，并加入了一些预期的错误格式，可根据实际运用情况加入更多错误模板，具体使用方式clone本仓库代码，可直接在本地运行调试。
支持的tag标签：
| tag属性     | 说明                                              |
|---------|-------------------------------------------------|
| mapping | 将数据进行转义，一般数据表格中都是以明文输入输出数据，但在实际代码业务中只用0或1表示状态的场景。例：`mapping(已完成:1,未完成:2)` |
| unique  | 标识唯一性，在同列中唯一。例：`unique(true)`                                                |
| date    | 时间格式化，该属性分两部分，第一个参数：该单元格的时间日期格式（注意该格式非在数据表格中明文显示的格式，要查看该单元格在excel中的时间格式设置）；第二个参数：将转化到的格式。例：`date(01-02-06,2006-01-02)`                                             |
| name    | 指定该属性所在字段绑定的excel列名称，一般情况下数据表格都会指定每一列代表的含义，该属性十分重要，也是映射数据的主要参考依据，需要保持excel列名跟name属性保持一致，例：`name(*名称)`                                                |

以上tag属性统一作用在 `excel` 标签下，如某个字段需要绑定多个属性，属性间用 `“;”` 分隔