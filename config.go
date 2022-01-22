package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type SolanaConfig struct {
	JsonRPCURL    string
	WebsocketURL  string
	KeypairPath   string
	AddressLabels map[string]string
	Commitment    string
}

type SolanaConfigInfo struct {
	Dir           string
	MainnetPath   string
	TestnetPath   string
	DevnetPath    string
	ConfigMain    SolanaConfig
	ConfigTestnet SolanaConfig
	ConfigDevnet  SolanaConfig
}
type PingConfig struct {
	Count       int
	Interval    int
	Timeout     int
	PerPingTime int64
}
type SolanaPing struct {
	PingExePath   string
	Report        PingConfig
	DataPoint1Min PingConfig
	PingSetup
}
type PingSetup struct {
	TxTimeout               int64
	WaitConfirmationTimeout int64
	StatusCheckTime         int64
}

type Slack struct {
	Clusters   []Cluster
	WebHook    string
	ReportTime int
}

type Config struct {
	UseGCloudDB           bool
	GCloudCredentialPath  string
	DBConn                string
	HostName              string
	ServerIP              string
	Logfile               string
	ReportClusters        []Cluster
	DataPoint1MinClusters []Cluster
	SolanaConfigInfo
	SolanaPing
	Slack
}

func loadConfig() Config {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic("loadConfig error:" + err.Error())
	}
	c := Config{}
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(userHome + "/.config/ping-api")
	v.ReadInConfig()
	v.AutomaticEnv()
	host, err := os.Hostname()
	if err != nil {
		c.HostName = ""
	}
	c.UseGCloudDB = v.GetBool("UseGCloudDB")
	c.GCloudCredentialPath = v.GetString("GCloudCredentialPath")
	c.DBConn = v.GetString("DBConn")
	c.HostName = host
	c.ServerIP = v.GetString("ServerIP")

	c.ReportClusters = []Cluster{}
	for _, e := range v.GetStringSlice("Clusters.Report") {
		c.ReportClusters = append(c.ReportClusters, Cluster(e))
	}
	c.DataPoint1MinClusters = []Cluster{}
	for _, e := range v.GetStringSlice("Clusters.DataPoint1Min") {
		c.DataPoint1MinClusters = append(c.DataPoint1MinClusters, Cluster(e))
	}
	c.Logfile = v.GetString("Logfile")
	c.SolanaConfigInfo = SolanaConfigInfo{
		Dir:         v.GetString("SolanaConfig.Dir"),
		MainnetPath: v.GetString("SolanaConfig.MainnetPath"),
		TestnetPath: v.GetString("SolanaConfig.TestnetPath"),
		DevnetPath:  v.GetString("SolanaConfig.DevnetPath"),
	}
	if len(c.SolanaConfigInfo.MainnetPath) > 0 {
		sConfig, err := ReadSolanaConfigFile(c.SolanaConfigInfo.Dir + c.SolanaConfigInfo.MainnetPath)
		if err != nil {
			log.Fatal(err)
		}
		c.SolanaConfigInfo.ConfigMain = sConfig
	}
	if len(c.SolanaConfigInfo.TestnetPath) > 0 {
		sConfig, err := ReadSolanaConfigFile(c.SolanaConfigInfo.Dir + c.SolanaConfigInfo.TestnetPath)
		if err != nil {
			log.Fatal(err)
		}
		c.SolanaConfigInfo.ConfigTestnet = sConfig
	}
	if len(c.SolanaConfigInfo.DevnetPath) > 0 {
		sConfig, err := ReadSolanaConfigFile(c.SolanaConfigInfo.Dir + c.SolanaConfigInfo.DevnetPath)
		if err != nil {
			log.Fatal(err)
		}
		c.SolanaConfigInfo.ConfigDevnet = sConfig
	}

	c.SolanaPing = SolanaPing{
		PingExePath: v.GetString("SolanaPing.PingExePath"),
		Report: PingConfig{
			Count:       v.GetInt("SolanaPing.Report.Count"),
			Interval:    v.GetInt("SolanaPing.Report.Inverval"),
			Timeout:     v.GetInt("SolanaPing.Report.Timeout"),
			PerPingTime: v.GetInt64("SolanaPing.Report.PerPingTime"),
		},
		DataPoint1Min: PingConfig{
			Count:       v.GetInt("SolanaPing.DataPoint1Min.Count"),
			Interval:    v.GetInt("SolanaPing.DataPoint1Min.Inverval"),
			Timeout:     v.GetInt("SolanaPing.DataPoint1Min.Timeout"),
			PerPingTime: v.GetInt64("SolanaPing.DataPoint1Min.PerPingTime"),
		},
	}
	c.SolanaPing.PingSetup.TxTimeout = v.GetInt64("SolanaPing.PingSetup.TxTimeout")
	c.SolanaPing.PingSetup.WaitConfirmationTimeout = v.GetInt64("SolanaPing.PingSetup.WaitConfirmationTimeout")
	c.SolanaPing.PingSetup.StatusCheckTime = v.GetInt64("SolanaPing.PingSetup.StatusCheckTime")
	sCluster := []Cluster{}
	for _, e := range v.GetStringSlice("Slack.Clusters") {
		sCluster = append(sCluster, Cluster(e))
	}
	c.Slack = Slack{
		Clusters:   sCluster,
		WebHook:    v.GetString("Slack.WebHook"),
		ReportTime: v.GetInt("Slack.ReportTime"),
	}
	osPath := os.Getenv("PATH")
	if len(osPath) != 0 {
		osPath = c.PingExePath + ":" + osPath
		os.Setenv("PATH", osPath)
	}
	os.Setenv("PATH", c.PingExePath)
	os.Setenv("PATH", c.PingExePath)
	gcloudCredential := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if len(gcloudCredential) == 0 && len(c.GCloudCredentialPath) != 0 {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.GCloudCredentialPath)
	}
	return c
}

func ReadSolanaConfigFile(filepath string) (SolanaConfig, error) {
	configmap := make(map[string]string, 1)
	addressmap := make(map[string]string, 1)

	f, err := os.Open(filepath)
	if err != nil {
		log.Printf("error opening file: %v\n", err)
		return SolanaConfig{}, err
	}
	r := bufio.NewReader(f)
	line, _, err := r.ReadLine()
	for err == nil {
		k, v := ToKeyPair(string(line))
		if k == "address_labels" {
			line, _, err := r.ReadLine()
			lKey, lVal := ToKeyPair(string(line))
			if err == nil && string(line)[0:1] == " " {
				if len(lKey) > 0 && len(lVal) > 0 {
					addressmap[lKey] = lVal
				}
			} else {
				configmap[k] = v
			}
		} else {
			configmap[k] = v
		}

		line, _, err = r.ReadLine()
	}
	return SolanaConfig{
		JsonRPCURL:    configmap["json_rpc_url"],
		WebsocketURL:  configmap["websocket_url:"],
		KeypairPath:   configmap["keypair_path"],
		AddressLabels: addressmap,
		Commitment:    configmap["commitment"],
	}, nil
}

func ToKeyPair(line string) (key string, val string) {
	noSpaceLine := strings.TrimSpace(string(line))
	idx := strings.Index(noSpaceLine, ":")
	if idx == -1 || idx == 0 { // not found or only have :
		return "", ""
	}
	if (len(noSpaceLine) - 1) == idx { // no value
		return strings.TrimSpace(noSpaceLine[0:idx]), ""
	}
	return strings.TrimSpace(noSpaceLine[0:idx]), strings.TrimSpace(noSpaceLine[idx+1:])
}
