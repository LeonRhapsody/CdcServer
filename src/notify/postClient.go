package notify

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

var GlobalTransport *http.Transport

//应该定义一个全局的 transport 结构体, 在多个 goroutine 之间共享.否则会占用大量的open files，引发socket: too many open files

func init() { //忽略证书检验
	GlobalTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//ResponseHeaderTimeout: 2 * time.Second,//限制读取response header的时间
		Dial: (&net.Dialer{
			Timeout: 10 * time.Second, //限制建立tcp连接的时间
			//KeepAlive: 30 * time.Second,
		}).Dial,
	}

}

type TaskInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Task []struct {
		SysID     string `json:"SysID"`
		Type      string `json:"Type"`
		Status    string `json:"Status"`
		Name      string `json:"Name"`
		CheckLag  string `json:"CheckLag"`
		CommitLag string `json:"CommitLag"`
		Error     string `json:"Error"`
	}
}

type Alert struct {
	AlarmTypeCode string      `json:"alarmTypeCode"`  //告警类型
	AlarmObject   string      `json:"alarmObject"`    //告警对象
	AlarmCatCode  string      `json:"alarmCatCode"`   //告警类型编码
	AlarmContent  string      `json:"alarmContent"`   //告警事件内容
	AlarmLevel    string      `json:"alarmLevel"`     //告警等级，0：紧急；1：重要；2：次要；3：提示
	AlarmTime     string      `json:"alarmTime"`      //告警事件
	AlarmAddress  string      `json:"alarmAddress"`   //告警地址
	AlarmDataType string      `json:"alarmData_type"` //告警数据类型，0：文本；1：JSON
	AlarmData     string      `json:"alarmData"`      //告警数据
	RepairType    string      `json:"repairType"`     //是否可修复，0：否，1：是
	AlarmDsType   interface{} `json:"alarmDsType"`    //告警数据源类型，0：业务数据库；1：数据湖；2：纯净湖
	AlarmSysCode  interface{} `json:"alarmSysCode"`   //告警系统编码
	AlarmTabName  interface{} `json:"alarmTabName"`   //告警表名称
}

type CollectedData struct {
	Status    string
	Type      string
	Name      string
	SysID     string
	Trace     string
	CommitLag string
	CheckLag  string
	IP        string
}
type ClientInfo struct {
	IP   string
	Port string
	Type string
}

func oggToNotifyData(c CollectedData) ([]byte, error) {
	now := time.Now()
	var a Alert
	a.AlarmTypeCode = "JHIDC_CDC_LOG_APP_EXIT"
	a.AlarmCatCode = "JHOM"
	a.AlarmObject = "OGG故障"
	a.AlarmContent = c.SysID + "系统OGG进程意外暂停，请立即排查! NAME: " + c.Name
	a.AlarmLevel = "0"
	a.AlarmTime = now.Format("2006-01-02 15:04:05")
	a.AlarmAddress = c.IP
	a.AlarmDataType = "1"
	a.AlarmData = c.Trace
	a.AlarmDsType = "0"
	a.AlarmSysCode = c.SysID
	a.AlarmTabName = ""
	jsonBytes, err := json.Marshal(a) //结构体转json序列化
	if err != nil {
		return nil, fmt.Errorf("告警json串解析异常")
	}
	return jsonBytes, nil

}

func Notify(uri string, data []byte) error {

	body := bytes.NewBuffer([]byte(data))
	res, err := http.Post(uri, "application/json;charset=utf-8", body)
	if err != nil {
		return fmt.Errorf("告警信息请求发送异常,%v", err)
	}
	defer res.Body.Close()

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("告警发送请求响应异常,%v", err)
	}

	//str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	return nil

}

// OggInfoToCollectedData 将status解析成单独的进程告警
func (t TaskInfo) OggInfoToCollectedData(i int) CollectedData {

	var cd CollectedData
	cd.Status = t.Task[i].Status
	cd.Type = t.Task[i].Type
	cd.Name = t.Task[i].Name
	cd.Trace = t.Task[i].Error
	cd.SysID = t.Task[i].SysID
	cd.CommitLag = t.Task[i].CommitLag
	cd.CheckLag = t.Task[i].CheckLag
	cd.IP = t.IP

	return cd

}

func (c ClientInfo) GetStatus() (TaskInfo, error) {
	//log.Printf("[%s] 获取状态", c.IP)
	content, err := c.GetFromClient("status")
	if err != nil {
		return TaskInfo{}, err
	}

	var t TaskInfo
	err = json.Unmarshal(content, &t)
	if err != nil {
		return TaskInfo{}, err
	}
	return t, err
}

func (c ClientInfo) RestartTask(name string) (string, error) {
	result, err := c.GetFromClient("restart/" + name)
	return string(result), err
}

// Reset 复位整个ogg进程组，可以解决服务器重启、数据库重启、进程意外被杀、pum通信故障
func (c ClientInfo) Reset() (string, error) {
	result, err := c.GetFromClient("repair")
	return string(result), err
}

func (c ClientInfo) GetFromClient(path string) ([]byte, error) {
	client := http.Client{Transport: GlobalTransport} //忽略证书检验

	uri := "https://" + c.IP + ":" + c.Port + "/ogg/" + path
	res, err := client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(res.Body)

	time.Sleep(10 * time.Millisecond)
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
	}

	//str := (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	return content, err

}

// UpdateRepDefFile 更新复制进程的表结构定义文件，可以解决便结构不一致的情况
func (c ClientInfo) UpdateRepDefFile(from string, to string) (string, error) {
	result, err := c.GetFromClient("UpdateRepDefFile/" + from + "/" + to)
	return string(result), err
}
