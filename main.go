package main

import (
	"github.com/LeonRhapsody/CdcServer/src/config"
	"github.com/LeonRhapsody/CdcServer/src/notify"
	"github.com/LeonRhapsody/CdcServer/src/shell"
	"github.com/LeonRhapsody/CdcServer/src/webhook"
	"github.com/gin-gonic/gin"
	"log"
	"sync"
)

func HttpCenter(runConfig config.RunConfig) {
	gin.SetMode(gin.ReleaseMode)

	route := gin.Default()
	route.SetTrustedProxies([]string{"192.168.1.2"})

	notifyAddress := webhook.NotifyAddress{
		Uri: runConfig.NotifyAddress,
	}

	//DownloadFileControl
	upgrade := route.Group("upgrade")
	{
		upgrade.GET("/:filename", func(c *gin.Context) {
			filename := c.Param("filename")
			filepath := "file/" + filename
			c.File(filepath)
		})
		upgrade.HEAD("/:filename", func(c *gin.Context) {
			filename := c.Param("filename")
			filepath := "file/" + filename
			c.File(filepath)

		})

	}

	//Post
	oggGroup := route.Group("ogg")
	{
		oggGroup.POST("/post", notifyAddress.Post)
	}

	listener := runConfig.Address + ":" + runConfig.Port
	key := runConfig.Ssl.Key
	crt := runConfig.Ssl.Cert

	err := route.RunTLS(listener, crt, key)
	if err != nil {
		log.Println(err)
	}

}

func main() {


	runConfig := config.ReadConfig("conf/config.yaml")


	fatherMap:=make(map[string]string)
	for _, client := range runConfig.SrcNodes {
		fatherID:=shell.Awk(client,":")[0]
		fatherAddress:=shell.Awk(client,":")[1]+":"+shell.Awk(client,":")[2]
		fatherMap[fatherID]=fatherAddress
	}


	notifyAddress := runConfig.NotifyAddress

	clientList, err := runConfig.ConfigToClients()

	if err != nil {
		log.Println(err)
		return
	}

	for _, c := range clientList {
		w := notify.WatchDog{
			Mu:               new(sync.RWMutex),
			Client:           c,
			RetryTimes:       make(map[string]int),
			DelayTimes:       make(map[string]int),
			NeedNotifyCh:     make(chan []byte, 1),
			RepairCh:         make(chan notify.CollectedData, 1),
			CheckInterval:    runConfig.CheckInterval,
			NotifyInhibition: runConfig.NotifyInhibition,
			RepairTimes:      runConfig.RepairTimes,
		}

		if err != nil {
			log.Println(err)
		}
		go w.Send()
		go w.Repair(fatherMap)
		go w.SendNotify(notifyAddress)

	}

	HttpCenter(runConfig)

}
