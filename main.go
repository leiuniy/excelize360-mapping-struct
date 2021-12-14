package main

import (
	"bytes"
	"encoding/json"
	"example.com/m/v2/excel"
	"github.com/astaxie/beego/validation"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"time"
)

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

func main() {
	file, err := ioutil.ReadFile("./test-success.xlsx")
	if err != nil {
		log.Println(err.Error())
		return
	}
	p := new(Project)
	p.Time = time.Now()
	processor, err := excel.NewProcessor(p, true)
	if err != nil {
		log.Println(err.Error())
		return
	}
	res, err := processor.ParseContent(bytes.NewReader(file), 1, 2)
	if err != nil {
		log.Println(err)
		return
	}
	errList, has := res.HasError()
	if has {
		for k, v := range errList {
			log.Println(k, v)
		}
		return
	}
	projects := make([]Project, 0)
	err = res.Format(&projects)
	if err != nil {
		log.Println(err)
		return
	}
	marshal, err := json.Marshal(projects)
	log.Println(string(marshal))
	log.Println(projects)
}
