package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const applicationName string = "tasmota-proxy"
const applicationVersion = "v0.0.4"
const applicationUrl string = "https://github.com/smford/tasmota-proxy"

var (
	apikey      string
	verbose     bool
	homeDirName string

	commandList = map[string]string{
		"on":        `Power%20On`,
		"off":       `Power%20Off`,
		"status":    `Status0`,
		"statusall": `Status0`,
	}
)

// structure of responses to status
type StatusResponse struct {
	Status struct {
		Module       int      `json:"Module"`
		DeviceName   string   `json:"DeviceName"`
		FriendlyName []string `json:"FriendlyName"`
		Topic        string   `json:"Topic"`
		ButtonTopic  string   `json:"ButtonTopic"`
		Power        int      `json:"Power"`
		PowerOnState int      `json:"PowerOnState"`
		LedState     int      `json:"LedState"`
		LedMask      string   `json:"LedMask"`
		SaveData     int      `json:"SaveData"`
		SaveState    int      `json:"SaveState"`
		SwitchTopic  string   `json:"SwitchTopic"`
		SwitchMode   []int    `json:"SwitchMode"`
		ButtonRetain int      `json:"ButtonRetain"`
		SwitchRetain int      `json:"SwitchRetain"`
		SensorRetain int      `json:"SensorRetain"`
		PowerRetain  int      `json:"PowerRetain"`
		InfoRetain   int      `json:"InfoRetain"`
		StateRetain  int      `json:"StateRetain"`
	} `json:"Status"`
	StatusPRM struct {
		Baudrate      int    `json:"Baudrate"`
		SerialConfig  string `json:"SerialConfig"`
		GroupTopic    string `json:"GroupTopic"`
		OtaURL        string `json:"OtaUrl"`
		RestartReason string `json:"RestartReason"`
		Uptime        string `json:"Uptime"`
		StartupUTC    string `json:"StartupUTC"`
		Sleep         int    `json:"Sleep"`
		CfgHolder     int    `json:"CfgHolder"`
		BootCount     int    `json:"BootCount"`
		BCResetTime   string `json:"BCResetTime"`
		SaveCount     int    `json:"SaveCount"`
		SaveAddress   string `json:"SaveAddress"`
	} `json:"StatusPRM"`
	StatusFWR struct {
		Version       string `json:"Version"`
		BuildDateTime string `json:"BuildDateTime"`
		Boot          int    `json:"Boot"`
		Core          string `json:"Core"`
		Sdk           string `json:"SDK"`
		CPUFrequency  int    `json:"CpuFrequency"`
		Hardware      string `json:"Hardware"`
		Cr            string `json:"CR"`
	} `json:"StatusFWR"`
	StatusLOG struct {
		SerialLog  int      `json:"SerialLog"`
		WebLog     int      `json:"WebLog"`
		MqttLog    int      `json:"MqttLog"`
		SysLog     int      `json:"SysLog"`
		LogHost    string   `json:"LogHost"`
		LogPort    int      `json:"LogPort"`
		SSID       []string `json:"SSId"`
		TelePeriod int      `json:"TelePeriod"`
		Resolution string   `json:"Resolution"`
		SetOption  []string `json:"SetOption"`
	} `json:"StatusLOG"`
	StatusMEM struct {
		ProgramSize      int      `json:"ProgramSize"`
		Free             int      `json:"Free"`
		Heap             int      `json:"Heap"`
		ProgramFlashSize int      `json:"ProgramFlashSize"`
		FlashSize        int      `json:"FlashSize"`
		FlashChipID      string   `json:"FlashChipId"`
		FlashFrequency   int      `json:"FlashFrequency"`
		FlashMode        int      `json:"FlashMode"`
		Features         []string `json:"Features"`
		Drivers          string   `json:"Drivers"`
		Sensors          string   `json:"Sensors"`
	} `json:"StatusMEM"`
	StatusNET struct {
		Hostname   string  `json:"Hostname"`
		IPAddress  string  `json:"IPAddress"`
		Gateway    string  `json:"Gateway"`
		Subnetmask string  `json:"Subnetmask"`
		DNSServer1 string  `json:"DNSServer1"`
		DNSServer2 string  `json:"DNSServer2"`
		Mac        string  `json:"Mac"`
		Webserver  int     `json:"Webserver"`
		HTTPAPI    int     `json:"HTTP_API"`
		WifiConfig int     `json:"WifiConfig"`
		WifiPower  float64 `json:"WifiPower"`
	} `json:"StatusNET"`
	StatusMQT struct {
		MqttHost       string `json:"MqttHost"`
		MqttPort       int    `json:"MqttPort"`
		MqttClientMask string `json:"MqttClientMask"`
		MqttClient     string `json:"MqttClient"`
		MqttUser       string `json:"MqttUser"`
		MqttCount      int    `json:"MqttCount"`
		MaxPacketSize  int    `json:"MAX_PACKET_SIZE"`
		Keepalive      int    `json:"KEEPALIVE"`
		SocketTimeout  int    `json:"SOCKET_TIMEOUT"`
	} `json:"StatusMQT"`
	StatusTIM struct {
		Utc      string `json:"UTC"`
		Local    string `json:"Local"`
		StartDST string `json:"StartDST"`
		EndDST   string `json:"EndDST"`
		Timezone int    `json:"Timezone"`
		Sunrise  string `json:"Sunrise"`
		Sunset   string `json:"Sunset"`
	} `json:"StatusTIM"`
	StatusSNS struct {
		Time    string `json:"Time"`
		Switch1 string `json:"Switch1"`
	} `json:"StatusSNS"`
	StatusSTS struct {
		Time      string `json:"Time"`
		Uptime    string `json:"Uptime"`
		UptimeSec int    `json:"UptimeSec"`
		Heap      int    `json:"Heap"`
		SleepMode string `json:"SleepMode"`
		Sleep     int    `json:"Sleep"`
		LoadAvg   int    `json:"LoadAvg"`
		MqttCount int    `json:"MqttCount"`
		Power     string `json:"POWER"`
		Wifi      struct {
			Ap        int    `json:"AP"`
			SSID      string `json:"SSId"`
			BSSID     string `json:"BSSId"`
			Channel   int    `json:"Channel"`
			Mode      string `json:"Mode"`
			Rssi      int    `json:"RSSI"`
			Signal    int    `json:"Signal"`
			LinkCount int    `json:"LinkCount"`
			Downtime  string `json:"Downtime"`
		} `json:"Wifi"`
	} `json:"StatusSTS"`
}

// structure of responses from poweron, poweroff
type PowerResponse struct {
	Power string `json:"POWER"`
}

func init() {

	homeDirName, err := os.UserHomeDir()
	checkErr(err)

	flag.String("cmd", "", "Command")
	flag.String("config", homeDirName+"/.tasproxy", "Configuration file: /path/to/file.yaml, default = "+homeDirName+"/.tasproxy")
	flag.String("device", "", "Device")
	flag.Bool("displayconfig", false, "Display configuration")
	flag.Bool("help", false, "Help")
	flag.String("ip", "", "IP")
	flag.Bool("list", false, "List Devices")
	flag.Bool("version", false, "Version")

	// temp
	flag.Bool("verbose", false, "Be verbose")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err = viper.BindPFlags(pflag.CommandLine)

	checkErr(err)

	viper.SetEnvPrefix("TASPROXY")
	err = viper.BindEnv("config")
	checkErr(err)

	if viper.GetBool("help") {
		displayHelp()
		os.Exit(0)
	}

	if viper.GetBool("version") {
		fmt.Println(applicationName + " " + applicationVersion)
		os.Exit(0)
	}

	configdir, configfile := filepath.Split(viper.GetString("config"))

	// set default configuration directory to current directory
	if configdir == "" {
		configdir = "."
	}

	viper.SetConfigType("yaml")
	viper.AddConfigPath(configdir)

	config := strings.TrimSuffix(configfile, ".yaml")
	config = strings.TrimSuffix(config, ".yml")

	viper.SetConfigName(config)

	err = viper.ReadInConfig()
	checkErr(err)

	if viper.GetBool("displayconfig") {
		displayConfig()
		os.Exit(0)
	}

	if viper.GetBool("list") {
		displayDevices()
		os.Exit(0)
	}
}

func main() {
	// temp
	verbose = viper.GetBool("verbose")

	if !viper.IsSet("cmd") {
		fmt.Println("Error, no command defined")
		os.Exit(1)
	}

	myCommand := viper.GetString("cmd")

	// check if device is valid
	if checkDeviceValid(viper.GetString("device")) {
		if verbose {
			fmt.Printf("Device: %s found\n", viper.GetString("device"))
		}
	} else {
		fmt.Printf("Device: %s not found\n", viper.GetString("device"))
		os.Exit(1)
	}

	// check if command is valid
	if !isCommandValid(myCommand) {
		fmt.Printf("Command \"%s\" is invalid\n", myCommand)
		os.Exit(1)
	}

	// send the command
	ipofdevice := viper.GetStringMap("devices")[viper.GetString("device")].(string)

	response, success := sendTasmota(ipofdevice, commandList[myCommand])

	if !success {
		// could not send to tasmota
		fmt.Println("oops not successful")
		os.Exit(1)
	} else {
		// good response from tasmota returned

		cleanCommand := strings.ToLower(myCommand)

		// if power on or power off
		if strings.EqualFold(cleanCommand, "on") || strings.EqualFold(cleanCommand, "off") {
			res := PowerResponse{}
			err := json.Unmarshal(response, &res)
			checkErr(err)
			fmt.Printf("%s:%s\n", viper.GetString("device"), res.Power)
		}

		// if status or statusall
		if strings.EqualFold(cleanCommand, "status") || strings.EqualFold(cleanCommand, "statusall") {
			res := StatusResponse{}
			err := json.Unmarshal(response, &res)
			checkErr(err)

			if strings.EqualFold(cleanCommand, "status") {
				var powerState string
				switch res.Status.Power {
				case 0:
					powerState = "OFF"
				case 1:
					powerState = "ON"
				default:
					powerState = "UNKNOWN"
				}

				fmt.Printf("%s:%s\n", viper.GetString("device"), powerState)

			}

			if strings.EqualFold(cleanCommand, "statusall") {
				fmt.Printf("%s\n", prettyPrint(res))
			}

		}

	}
}

// send a command to the tasmota
func sendTasmota(ip string, cmd string) ([]byte, bool) {

	url := fmt.Sprintf("http://%s/cm?cmnd=%s", ip, cmd)

	if verbose {
		fmt.Printf("URL: %s\n", url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
	}

	client := &http.Client{}
	client.Timeout = time.Second * 5
	resp, err := client.Do(req)
	checkErr(err)

	defer resp.Body.Close()

	wassucessful := false
	if resp.StatusCode == http.StatusOK {
		wassucessful = true
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	checkErr(err)

	return bodyBytes, wassucessful

}

// prints out json pretty
func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

// checks if a command is valid
func isCommandValid(command string) bool {
	if _, ok := commandList[command]; ok {
		return true
	}

	return false
}

// display help
func displayHelp() {
	message := `
      --cmd [x]             Default line character (default: = )
      --config [file]       Configuration file: /path/to/file.yaml, default = "` + homeDirName + `"/.tasproxy"
      --device [name]       Name of device
      --displayconfig       Display configuration
      --ip [ip address]     IP address
      --list                List all configured devices
      --help                Display help
      --version             Display version`
	fmt.Println(applicationName + " " + applicationVersion + "\n" + applicationUrl)
	fmt.Println(message)
}

// captures and prints errors
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// display configuration
func displayConfig() {
	allmysettings := viper.AllSettings()
	var keys []string
	for k := range allmysettings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Println("CONFIG:", k, ":", allmysettings[k])
	}
}

// list devices
func displayDevices() {
	if viper.IsSet("devices") {
		for k, v := range viper.GetStringMap("devices") {
			fmt.Printf("%s     %s\n", k, v)
		}
	} else {
		fmt.Println("no devices found")
	}
}

// check that the device exists
func checkDeviceValid(device string) bool {
	if _, ok := viper.GetStringMap("devices")[device]; ok {
		return true
	} else {
		// device isn't found
		return false
	}
}
