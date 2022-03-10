package main

import (
	"encoding/json"
	"example.com/m/v2/excel"
	"github.com/astaxie/beego/validation"
	"log"
	"os"
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
	log.Println(p.Statistics.Num, p.Result.Count)
	valid := validation.Validation{}
	valid.Match(p.Name, regexp.MustCompile("^[-_a-zA-Z一-龥]{1,64}$"), strconv.Itoa(excel.ParamUnqualified))
	for _, err := range valid.Errors {
		errType, _ := strconv.Atoi(err.Key)
		res.AddError(errType, "名称")
	}
	return nil
}

type VehicleRepair struct {
	Total                   string `json:"total" excel:"name(Total)"`
	A                       string `json:"a" excel:"name(A)"`
	B                       string `json:"b" excel:"name(B)"`
	ProdNo                  string `json:"prod_no" excel:"name(ProdNo)"`
	VehType                 string `json:"veh_type" excel:"name(Veh. type)"`
	CaptTimeSDateAMin       string `json:"capt_time_s_date_a_min" excel:"name(Capt. time S Date a. Min.)"`
	CaptChkptS              string `json:"capt_chkpt_s" excel:"name(Capt. chkpt. S)"`
	CaptChkptR              string `json:"capt_chkpt_r" excel:"name(Capt. chkpt. R)"`
	CausingCostCtrS         string `json:"causing_cost_ctr_s" excel:"name(Causing cost ctr S)"`
	CausingCostCtrGrpSGroup string `json:"causing_cost_ctr_grp_s_group" excel:"name(Causing cost ctr grp S Group)"`
	CaptureTimeRDateAMin    string `json:"capture_time_r_date_a_min" excel:"name(Capture time R Date a. Min.)"`
	CurrentCheckpoint       string `json:"current_checkpoint" excel:"name(Current checkpoint)"`
	CurrentDateDateAMin     string `json:"current_date_date_a_min" excel:"name(Current date Date a. Min.)"`
	CurrentShiftNumber      string `json:"current_shift_number" excel:"name(Current shift number)"`
	FaultLocationS          string `json:"fault_location_s" excel:"name(Fault location S)"`
	FaultTypeS              string `json:"fault_type_s" excel:"name(Fault type S)"`
	FaultPrioS              string `json:"fault_prio_s" excel:"name(Fault Prio S)"`
	BodyNo                  string `json:"body_no" excel:"name(BodyNo)"`
	BaumusterSalesDesc      string `json:"baumuster_sales_desc" excel:"name(Baumuster Sales desc.)"`
	OperatorR               string `json:"operator_r" excel:"name(Operator R)"`
	StampNoCapturerR        string `json:"stamp_no_capturer_r" excel:"name(StampNo capturer R)"`
	Tenant                  string `json:"tenant"`
	CreatedUser             string `json:"created_user"`
	UpdatedUser             string `json:"updated_user"`
}

func main() {
	file, err := os.Open("./Report.xls")
	if err != nil {
		log.Println(err.Error())
		return
	}
	p := new(VehicleRepair)
	//p := new(Project)
	//p.Time = time.Now()
	processor, err := excel.NewProcessor(p, false)
	if err != nil {
		log.Println(err.Error())
		return
	}
	// Report.xls (2,4) test-success.xlsx(1,2)
	res, err := processor.ParseContent(file, 2, 4)
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
	projects := make([]VehicleRepair, 0)
	//projects := make([]Project, 0)
	err = res.Format(&projects)
	if err != nil {
		log.Println(err)
		return
	}
	marshal, err := json.Marshal(projects)
	log.Println(string(marshal))
	log.Println(projects)
}
