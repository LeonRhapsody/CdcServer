package notify

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
)

type WatchDog struct {
	Mu               *sync.RWMutex
	Client           ClientInfo
	RetryTimes       map[string]int
	DelayTimes       map[string]int
	NeedNotifyCh     chan []byte
	RepairCh         chan CollectedData
	CheckInterval    time.Duration
	NotifyInhibition int
	RepairTimes      int
}

// Repair 修复已知问题
func (w WatchDog) Repair(fatherMap map[string]string) {


	log.Printf("[%s] start repair thread", w.Client.IP)
	for s := range w.RepairCh {

		w.Mu.Lock()
		w.RetryTimes[s.IP+s.Name]++ //进来就加1
		w.Mu.Unlock()
		log.Printf("[%s] try repair %s,type is %s,error message is %s", w.Client.IP, s.Name, s.Type, s.Trace)

		if s.Status == "STOPPED" {
			w.Client.Reset()
			log.Printf("cover %s error,reset ogg status,", s.Name)
		} else {
			switch s.Type {
			case "MANAGER":
				w.Client.Reset()
				log.Printf("cover %s error,reset ogg status,", s.Name)
			case "PUM":
				if strings.Contains(s.Trace, " TCP/IP ") {
					w.Client.Reset()
					log.Printf("cover %s error,reset ogg status,", s.Name)
				}
			case "EXTRACT":
			case "REP":
				if strings.Contains(s.Trace, "Bad column") {
					father:=fatherMap[s.SysID]
					w.Client.UpdateRepDefFile(father, s.Name)
					log.Printf("cover %s error,update ogg struct define files,", s.Name)
					w.Client.Reset()
					log.Printf("cover %s error,reset ogg status,", s.Name)
				} else if strings.Contains(s.Trace, "aaaJava or JNI exception") {
					w.Client.Reset()
					log.Printf("cover %s error,reset ogg status,", s.Name)
				}
			default:
				return
			}
		}

	}

}

// Send 将不正常的task 信息发送到ch
func (w WatchDog) Send() {

	log.Printf("[%s] start check thread", w.Client.IP)
	ticker := time.NewTicker(w.CheckInterval) //压测


	var cd CollectedData

	for range ticker.C {

		log.Printf("[%s] 获取状态", w.Client.IP)
		t, err := w.Client.GetStatus()
		if err != nil {
			log.Printf("[%s] 获取状态失败，发出告警，错误原因：%s", w.Client.IP, err)
			var a Alert
			a.AlarmTypeCode = "JHOM_CLIENT_ABNORMALITY"
			a.AlarmCatCode = "JHOM"
			a.AlarmObject = "watchDog is Down"
			a.AlarmContent = w.Client.IP + " watchDog is Down"
			a.AlarmLevel = "0"
			a.AlarmTime = time.Now().Format("2006-01-02 15:04:05")
			a.AlarmAddress = w.Client.IP
			a.AlarmDataType = "1"
			a.AlarmData = w.Client.IP + " watchDog is Down"
			a.AlarmDsType = "0"
			a.AlarmSysCode = ""
			a.AlarmTabName = ""
			jsonBytes, _ := json.Marshal(a) //结构体转json序列化
			w.NeedNotifyCh <- jsonBytes

		}

		for i := 0; i < len(t.Task); i++ {

			if t.Task[i].Status != "RUNNING" { //状态不正常触发
				cd = t.OggInfoToCollectedData(i)
				log.Printf("[%s] %s is not working", w.Client.IP, t.Task[i].Name)

				if w.RetryTimes[t.IP+t.Task[i].Name] < w.RepairTimes { //repair2
					w.RepairCh <- cd
					log.Printf("[%s] send %s to repair", w.Client.IP, t.Task[i].Name)
					log.Printf("[%s] 第%d次修复", w.Client.IP, w.RetryTimes[t.IP+t.Task[i].Name])
				} else { //修复三次仍不成功，发出告警
					log.Printf("[%s] 第%d次delay", w.Client.IP, w.DelayTimes[t.IP+t.Task[i].Name])
					w.DelayTimes[t.IP+t.Task[i].Name]++ //修复三次仍不成功，开始计数delay,目的是为了降噪

					switch w.DelayTimes[t.IP+t.Task[i].Name] {
					case 1: //第一次，发送到告警队列
						notifyData, _ := oggToNotifyData(cd)
						w.NeedNotifyCh <- notifyData
						log.Printf("[%s] %s is repaired 3 times,but it still not work,send it to notify", w.Client.IP, t.Task[i].Name)

					case w.NotifyInhibition: //告警抑制
						w.DelayTimes[t.IP+t.Task[i].Name] = 0
						w.DelayTimes[t.IP+t.Task[i].Name] = 0
						w.RepairCh <- cd
					default:

					}

				}
			}

		}
	}
}

// SendNotify  读取异常队列，发送告警
func (w WatchDog) SendNotify(uri string) {
	for s := range w.NeedNotifyCh {
		err := Notify(uri, s)
		if err != nil {
			log.Println(err)
		}
	}
}
