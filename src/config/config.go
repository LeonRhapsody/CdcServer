package config

import (
	"fmt"
	"github.com/LeonRhapsody/CdcServer/src/notify"
	"github.com/LeonRhapsody/CdcServer/src/shell"
	"github.com/spf13/viper"
	"time"
)

type RunConfig struct {
	Address          string        `yaml:"address"`
	ExecFilename     string        `yaml:"execFilename"`
	Ssl              Ssl           `yaml:"ssl"`
	Port             string        `yaml:"port"`
	NotifyAddress    string        `yaml:"notifyAddress"`
	CheckInterval    time.Duration `yaml:"checkInterval"`
	NotifyInhibition int           `yaml:"notifyInhibition"`
	RepairTimes      int           `yaml:"repairTimes"`
	SrcNodes         []string      `yaml:"srcNodes"`
	DesNodes         []string      `yaml:"desNodes"`
}

type Ssl struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	Ca   string `yaml:"ca"`
}

func defaultConfig() {
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", "34568")
	viper.SetDefault("execFilename", "oggWatchDog")
	viper.SetDefault("cert", "./ogg.crt")
	viper.SetDefault("key", "./ogg.key")
	viper.SetDefault("ca", "./ca")
	viper.SetDefault("checkInterval", 5*time.Second)
	viper.SetDefault("notifyInhibition", 60)
	viper.SetDefault("repairTimes", 3)

}

func ReadConfig(configName string) RunConfig {
	defaultConfig()

	viper.SetConfigFile(configName)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	var runConfig RunConfig
	viper.Unmarshal(&runConfig)
	return runConfig
}


func (r RunConfig) ConfigToClients() ([]notify.ClientInfo, error) {
	var chunks []notify.ClientInfo

	for _, client := range r.SrcNodes {


			a := shell.Awk(client, ":")
			if len(a) != 3 {
				return nil, fmt.Errorf("文件格式不对")
			}

			n := notify.ClientInfo{
				IP:   a[1],
				Port: a[2],
			}
			chunks = append(chunks, n)

		}

	for _, client := range r.DesNodes {


		a := shell.Awk(client, ":")
		if len(a) != 2 {
			return nil, fmt.Errorf("文件格式不对")
		}

		n := notify.ClientInfo{
			IP:   a[0],
			Port: a[1],
		}
		chunks = append(chunks, n)

	}


	return chunks, nil

}
