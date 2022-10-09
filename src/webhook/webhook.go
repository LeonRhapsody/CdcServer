package webhook

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

type JhAlert struct {
	AlarmTypeCode string `json:"alarmTypeCode"`  //告警类型编码
	AlarmCatCode  string `json:"alarmCatCode"`   //告警类型编码
	AlarmObject   string `json:"alarmObject"`    //告警对象
	AlarmContent  string `json:"alarmContent"`   //告警事件内容
	AlarmLevel    string `json:"alarmLevel"`     //告警等级，0：紧急；1：重要；2：次要；3：提示
	AlarmTime     string `json:"alarmTime"`      //告警事件
	AlarmAddress  string `json:"alarmAddress"`   //告警地址
	AlarmDataType string `json:"alarmData_type"` //告警数据类型，0：文本；1：JSON
	AlarmData     string `json:"alarmData"`      //告警数据
	RepairType    string `json:"repairType"`     //是否可修复，0：否，1：是
	AlarmDsType   interface {
	} `json:"alarmDsType"` //告警数据源类型，0：业务数据库；1：数据湖；2：纯净湖
	AlarmSysCode interface {
	} `json:"alarmSysCode"` //告警系统编码
	AlarmTabName interface {
	} `json:"alarmTabName"` //告警表名称
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:annotations`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

type Notification struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:receiver`
	GroupLabels       map[string]string `json:groupLabels`
	CommonLabels      map[string]string `json:commonLabels`
	CommonAnnotations map[string]string `json:commonAnnotations`
	ExternalURL       string            `json:externalURL`
	Alerts            []Alert           `json:alerts`
}

type NotifyAddress struct {
	Uri string
}

func (n NotifyAddress) Post(c *gin.Context) {
	var notification Notification

	err := c.BindJSON(&notification)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ch := make(chan interface{})
	go TransformToMarkdown(notification, ch)
	n.send(ch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	}
	c.JSON(http.StatusOK, gin.H{"message": "send  successful!"})

}

func TransformToMarkdown(notification Notification, ch chan interface{}) {
	defer close(ch)

	for _, alert := range notification.Alerts {
		ja := &JhAlert{
			AlarmTypeCode: "JHOM_MACHINE_ABNORMALITY",
			AlarmCatCode:  "JHOM",
			AlarmObject:   alert.Annotations["summary"],
			AlarmContent:  alert.Annotations["summary"],
			AlarmLevel:    "0",
			AlarmTime:     time.Now().Format("2006-01-02 15:04:05"),
			AlarmAddress:  alert.Labels["instance"],
			AlarmDataType: "1",
			//AlarmData:     alert.StartsAt.Format("2006-01-02 15:04:05") + "\n" + alert.Annotations["description"] + "\n计算公式：  " + alert.Labels["expr"] + "\n值：  " + alert.Annotations["value"],
			AlarmData:    alert.StartsAt.Format("2006-01-02 15:04:05") + "\n" + alert.Annotations["description"],
			RepairType:   "0",
			AlarmDsType:  "",
			AlarmSysCode: "",
			AlarmTabName: "",
		}
		switch alert.Labels["level"] {
		case "critical":
			ja.AlarmLevel = "2"
		case "warning":
			ja.AlarmLevel = "1"
		case "emergency":
			ja.AlarmLevel = "0"
		}
		ch <- ja
	}

}

func (n NotifyAddress) send(ch chan interface{}) (err error) {

	for data := range ch {

		if err != nil {
			return
		}

		jsonBytes, err := json.Marshal(data) //结构体转json序列化
		log.Printf("[转发告警] %s", string(jsonBytes))
		if err != nil {
			break
		}
		req, err := http.NewRequest(
			"POST",
			n.Uri,
			bytes.NewBuffer(jsonBytes))

		if err != nil {
			break
		}

		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			break
		}

		resp.Body.Close()

		log.Printf("[转发告警]response Status: %s", resp.Status)
	}
	return

}
