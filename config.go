package main

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config Config
type Config struct {
	MOD       string            `json:"mod" yaml:"mod" mapstructure:"mod"`
	DB        DBConfig          `json:"database" yaml:"database" mapstructure:"database"`
	LogLevel  string            `json:"logger" yaml:"logger" mapstructure:"logger"`
	UDP       string            `json:"udp" yaml:"udp" mapstructure:"udp"`
	API       string            `json:"api" yaml:"api" mapstructure:"api"`
	Secret    string            `json:"secret" yaml:"secret" mapstructure:"secret"`
	Media     MediaServer       `json:"media" yaml:"media" mapstructure:"media"`
	Stream    Stream            `json:"stream" yaml:"stream" mapstructure:"stream"`
	Record    RecordCfg         `json:"record" yaml:"record" mapstructure:"record"`
	GB28181   sysInfo           `json:"gb28181" yaml:"gb28181" mapstructure:"gb28181"`
	Notify    map[string]string `json:"notify" yaml:"notify" mapstructure:"notify"`
	notifyMap map[string]string
}

type RecordCfg struct {
	FilePath  string `json:"filepath" yaml:"filepath" mapstructure:"filepath"`
	Expire    int    `json:"expire" yaml:"expire"  mapstructure:"expire"`
	Recordmax int    `json:"recordmax" yaml:"recordmax"  mapstructure:"recordmax"`
}

// Stream Stream
type Stream struct {
	HLS  bool `json:"hls" yaml:"hls" mapstructure:"hls"`
	RTMP bool `json:"rtmp" yaml:"rtmp" mapstructure:"rtmp"`
}

// MediaServer MediaServer
type MediaServer struct {
	RESTFUL string `json:"restful" yaml:"restful" mapstructure:"restful"`
	HTTP    string `json:"http" yaml:"http" mapstructure:"http"`
	WS      string `json:"ws" yaml:"ws" mapstructure:"ws"`
	RTMP    string `json:"rtmp" yaml:"rtmp" mapstructure:"rtmp"`
	RTSP    string `json:"rtsp" yaml:"rtsp" mapstructure:"rtsp"`
	RTP     string `json:"rtp" yaml:"rtp" mapstructure:"rtp"`
	Secret  string `json:"secret" yaml:"secret" mapstructure:"secret"`
}

var config *Config

func loadConfig() {
	viper.SetConfigType("yml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./")
	viper.SetDefault("logger", "debug")
	viper.SetDefault("udp", "0.0.0.0:5060")
	viper.SetDefault("api", "0.0.0.0:8090")
	viper.SetDefault("mod", "release")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalln("init config error:", err)
	}
	logrus.Infoln("init config ok")
	config = &Config{}
	err = viper.Unmarshal(&config)
	if err != nil {
		logrus.Fatalln("init config unmarshal error:", err)
	}
	logrus.Infof("config :%+v", config)
	level, _ := logrus.ParseLevel(config.LogLevel)
	logrus.SetLevel(level)
	InitDB(config.DB)
	config.MOD = strings.ToUpper(config.MOD)
	notifyMap := map[string]string{}
	if config.Notify != nil {
		for k, v := range config.Notify {
			if v != "" {
				notifyMap[strings.ReplaceAll(k, "_", ".")] = v
			}
		}
	}
	config.notifyMap = notifyMap
	if config.Record.Expire == 0 {
		config.Record.Expire = 7
	}
	if config.Record.Recordmax == 0 {
		config.Record.Expire = 600
	}
}
