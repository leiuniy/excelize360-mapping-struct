package excel

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"io"
	"mime/multipart"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

const (
	ParamError = iota
	ParamInvalid
	ParamUnqualified
	AlreadyExists
	NotExist
	NotInConfigurationItems
	TimeFormatError
	DataOutsideExpectedLimits
	CannotGTCurrentTime
	AlreadyBinding
	ParamCannotBeEmpty
)

var errFormatList = map[int]excelErr{
	ParamCannotBeEmpty:        {"%s参数不可为空"},
	ParamInvalid:              {"%s参数输入错误"},
	ParamUnqualified:          {"%s参数格式不正确"},
	AlreadyExists:             {"%s已存在"},
	NotExist:                  {"%s不存在"},
	NotInConfigurationItems:   {"%s不在配置项中"},
	TimeFormatError:           {"%s时间格式错误"},
	DataOutsideExpectedLimits: {"%s数据不在预期限制范围"},
	CannotGTCurrentTime:       {"%s不可大于当前时间"},
	AlreadyBinding:            {"%s已绑定"},
	ParamError:                {"%s参数验证失败"},
}

type excelErr struct {
	errInfo string
}

func (e excelErr) Error() string {
	return e.errInfo
}

type processor struct {
	file         *excelize.File
	fieldMapping map[string]map[string]string
	sheetName    string
	body         MappingStruct
	val          reflect.Value
	openValid    bool
	uniqueMap    map[int][]string
}

type MappingStruct interface {
	ExcelRowProcess(res *Result) error
}

func NewProcessor(body MappingStruct, isValid bool) (*processor, error) {
	p := new(processor)
	p.val = reflect.ValueOf(body)
	if p.val.Kind() != reflect.Ptr {
		return nil, errors.New("body must be pointer struct")
	}
	//p.val = val.Elem()
	//p.val = reflect.TypeOf(body)
	p.body = body
	p.openValid = isValid

	p.fieldMapping = make(map[string]map[string]string)
	//生成结构体与excel头映射关系
	p.generateMapping(p.val, "")
	return p, nil
}

func (p *processor) generateMapping(val reflect.Value, baseField string) {
	switch val.Kind() {
	case reflect.Struct:
	case reflect.Ptr:
		//当结构体指针或字段指针为空，则创建一个指针指向
		if val.IsNil() {
			newValue := reflect.New(val.Type().Elem())
			val = reflect.NewAt(val.Type().Elem(), unsafe.Pointer(newValue.Pointer()))
		}
		val = val.Elem()
		p.generateMapping(val, baseField)
		return
	default:
		return
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldName := typ.Field(i).Name
		if baseField != "" {
			fieldName = fmt.Sprintf("%s.%s", baseField, fieldName)
		}
		excel, ok := typ.Field(i).Tag.Lookup("excel")
		if !ok {

			//生成嵌套结构体的映射关系
			fieldVal := val.Field(i)
			p.generateMapping(fieldVal, fieldName)
			continue
		}
		m := map[string]string{"name": fieldName}

		rex := regexp.MustCompile(`mapping\((.*?)\)`)
		for _, i := range rex.FindAllStringSubmatch(excel, -1) {
			if len(i) >= 2 {
				m["mapping"] = i[1]
			}
		}

		rex = regexp.MustCompile(`unique\((.*?)\)`)
		for _, i := range rex.FindAllStringSubmatch(excel, -1) {
			if len(i) >= 2 {
				m["unique"] = i[1]
			}
		}

		rex = regexp.MustCompile(`date\((.*?)\)`)
		for _, i := range rex.FindAllStringSubmatch(excel, -1) {
			if len(i) >= 2 {
				m["date"] = i[1]
			}
		}

		rex = regexp.MustCompile(`name\((.*?)\)`)
		for _, i := range rex.FindAllStringSubmatch(excel, -1) {
			if len(i) >= 2 {
				p.fieldMapping[strings.TrimSpace(i[1])] = m
			}
		}
	}
}

func (p *processor) ParseContent(file io.Reader, mappingHeaderRow int, dataStartRow int) (*Result, error) {
	if mappingHeaderRow-1 < 0 {
		return nil, errors.New("no Excel mapping header position is specified")
	}
	if mappingHeaderRow >= dataStartRow {
		return nil, errors.New("mapping header row position cannot be greater than or equal to the beginning of the data row")
	}
	var err error
	p.file, err = excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}
	p.uniqueMap = make(map[int][]string)
	p.sheetName = p.file.GetSheetName(1)
	rows := p.file.GetRows(p.sheetName)
	if len(rows) < dataStartRow {
		return nil, errors.New("excel file valid data behavior is empty")
	}

	res := new(Result)
	res.errors = make(map[int][]string)
	res.mappingResults = make([]interface{}, 0)
	if err := p.rows(rows, mappingHeaderRow, dataStartRow, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (p *processor) rows(rows [][]string, mappingHeaderRow, dataStartRow int, res *Result) error {
	for rowIndex := dataStartRow - 1; rowIndex < len(rows); rowIndex++ {
		res.rowIndex = rowIndex
		errList := make([]string, 0)
		newBodyVal := reflect.New(p.val.Type().Elem())
		newBodyVal.Elem().Set(p.val.Elem())
		for colIndex, col := range rows[rowIndex] {
			if colIndex >= len(rows[mappingHeaderRow-1]) {
				continue
			}
			mappingHeader := rows[mappingHeaderRow-1][colIndex]
			//去除列的前后空格
			colVal := strings.TrimSpace(col)
			mappingField, ok := p.fieldMapping[strings.TrimSpace(mappingHeader)]
			if !ok {
				continue
			}
			// 列唯一性校验
			errList = append(errList, p.uniqueFormat(rows, mappingHeader, &colVal, rowIndex, colIndex, mappingField)...)
			//格式化时间
			errList = append(errList, p.dateFormat(mappingHeader, &colVal, mappingField)...)
			//值映射转换
			mappingErrList := p.mappingFormat(mappingHeader, &colVal, mappingField)
			errList = append(errList, mappingErrList...)
			if len(mappingErrList) != 0 {
				continue
			}
			//参数赋值
			errs, err := p.parseValue(newBodyVal, mappingField["name"], mappingHeader, colVal)
			if err != nil {
				return err
			}
			errList = append(errList, errs...)
		}
		if len(errList) != 0 {
			res.errors[rowIndex+1] = errList
		}
		p.body = newBodyVal.Interface().(MappingStruct)
		//自定义参数验证
		if err := p.definedValid(res); err != nil {
			return err
		}
		if _, ok := res.HasError(); ok {
			continue
		}
		res.mappingResults = append(res.mappingResults, p.body)
	}
	return nil
}

func (p *processor) definedValid(res *Result) error {
	if p.openValid {
		return p.body.ExcelRowProcess(res)
	}
	return nil
}

func (p *processor) uniqueFormat(rows [][]string, mappingHeader string, col *string, rowIndex, colIndex int, mappingField map[string]string) []string {
	errList := make([]string, 0)
	format, ok := mappingField["unique"]
	if !ok || format != "true" {
		return errList
	}
	_, ok = p.uniqueMap[colIndex]
	if !ok {
		p.uniqueMap[colIndex] = make([]string, 0)
		for index := 0; index < len(rows); index++ {
			if len(rows[index]) <= colIndex {
				p.uniqueMap[colIndex] = append(p.uniqueMap[colIndex], rows[index][0])
				continue
			}
			p.uniqueMap[colIndex] = append(p.uniqueMap[colIndex], rows[index][colIndex])
		}
	}
	cols := p.uniqueMap[colIndex]
	for i, val := range cols {
		if i != rowIndex && val != "" && val == *col {
			errList = append(errList, fmt.Sprintf("%s[%s]不可重复", mappingHeader, *col))
			break
		}
	}
	return errList
}

func (p *processor) dateFormat(mappingHeader string, col *string, mappingField map[string]string) []string {
	errList := make([]string, 0)
	format, ok := mappingField["date"]
	if !ok || format == "" {
		return errList
	}
	formats := strings.SplitN(format, ",", 2)
	if *col == "" || len(formats) != 2 {
		return errList
	}
	location, err := time.ParseInLocation(formats[0], *col, time.Local)
	if err != nil {
		errList = append(errList, fmt.Sprintf("%s单元格格式错误", mappingHeader))
		return errList
	}
	*col = location.Format(formats[1])
	return errList
}

func (p *processor) mappingFormat(mappingHeader string, col *string, mappingField map[string]string) []string {
	errList := make([]string, 0)
	format, ok := mappingField["mapping"]
	if !ok || format == "" {
		return errList
	}
	mappingValues := make(map[string]string)
	formatStr := strings.Split(format, ",")
	for _, format := range formatStr {
		n := strings.SplitN(format, ":", 2)
		if len(n) != 2 {
			continue
		}
		mappingValues[n[0]] = n[1]
	}
	val, ok := mappingValues[*col]
	if ok {
		*col = val
		return errList
	}
	errList = append(errList, fmt.Sprintf("%s单元格存在非法输入", mappingHeader))
	return errList
}

func (p *processor) parseValue(val reflect.Value, fieldAddr, mappingHeader, col string) ([]string, error) {
	errList := make([]string, 0)
	fields := strings.Split(fieldAddr, ".")
	if len(fields) == 0 {
		return errList, nil
	}
	for _, field := range fields {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		val = val.FieldByName(field)
		errs, err := p.parse(val, col, mappingHeader)
		if err != nil {
			return errList, err
		}
		errList = append(errList, errs...)
	}
	return errList, nil
}

func (p *processor) parse(val reflect.Value, col, mappingHeader string) ([]string, error) {
	errList := make([]string, 0)
	var err error
	switch val.Kind() {
	case reflect.String:
		val.SetString(col)
	case reflect.Bool:
		parseBool, err := strconv.ParseBool(col)
		if err != nil {
			errList = append(errList, fmt.Sprintf("%s单元格非法输入,参数非bool类型值", mappingHeader))
		}
		val.SetBool(parseBool)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var value int64
		if col != "" {
			value, err = strconv.ParseInt(col, 10, 64)
			if err != nil {
				errList = append(errList, fmt.Sprintf("%s单元格非法输入,参数非整形数值", mappingHeader))
			}
		}
		val.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var value uint64
		if col != "" {
			value, err = strconv.ParseUint(col, 10, 64)
			if err != nil {
				errList = append(errList, fmt.Sprintf("%s单元格非法输入,参数非整形数值", mappingHeader))
			}
		}
		val.SetUint(value)
	case reflect.Float32, reflect.Float64:
		var value float64
		if col != "" {
			value, err = strconv.ParseFloat(col, 64)
			if err != nil {
				errList = append(errList, fmt.Sprintf("%s单元格非法输入,参数非浮点型数值", mappingHeader))
			}
		}
		val.SetFloat(value)
	case reflect.Struct:
		return errList, nil
	case reflect.Ptr:
		//初始化指针
		value := reflect.New(val.Type().Elem())
		val.Set(value)
		var errs []string
		errs, err = p.parse(val.Elem(), col, mappingHeader)
		if err != nil {
			break
		}
		errList = append(errList, errs...)
	default:
		return errList, fmt.Errorf("excel column[%s] parseValue unsupported type[%v] mappings", mappingHeader, val.Kind().String())
	}
	return errList, nil
}

type Result struct {
	errors         map[int][]string
	mappingResults []interface{}
	rowIndex       int
}

func (r *Result) errorFormat(errType int) string {
	err, ok := errFormatList[errType]
	if !ok {
		err = errFormatList[ParamError]
	}
	return err.Error()
}

func (r *Result) AddError(errType int, args ...string) *Result {
	if _, ok := r.errors[r.rowIndex+1]; !ok {
		r.errors[r.rowIndex+1] = make([]string, 0)
	}
	r.errors[r.rowIndex+1] = append(r.errors[r.rowIndex+1], fmt.Sprintf(r.errorFormat(errType), strings.Join(args, "")))
	return r
}

func (r *Result) HasError() (map[int][]string, bool) {
	return r.errors, len(r.errors) != 0
}

func (r *Result) List() []interface{} {
	return r.mappingResults
}

func (r *Result) Format(array interface{}) error {
	marshal, err := json.Marshal(r.mappingResults)
	if err != nil {
		return err
	}
	return json.Unmarshal(marshal, &array)
}

//ValidateExcelSuffix .
func ValidateExcelSuffix(header *multipart.FileHeader) error {
	var allowExtMap = map[string]bool{
		".xlsx": true,
		".xls":  true,
	}
	//判断文件后缀
	if _, ok := allowExtMap[path.Ext(header.Filename)]; !ok {
		return fmt.Errorf("file request format error，support XLSX and XLS")
	}
	return nil
}

//ValidateExcelSize .
func ValidateExcelSize(size int64) error {
	if size > 500*1024 {
		return fmt.Errorf("file maximum size 500 kb")
	}
	return nil
}
