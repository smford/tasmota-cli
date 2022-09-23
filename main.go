package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const applicationName string = "tasmota-proxy"
const applicationVersion = "v0.0.6.1"
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
		"timers":    `Timers`,
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

// structure of all timers
type AllTimers struct {
	Timers string `json:"Timers"`
	Timer1 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer1"`
	Timer2 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer2"`
	Timer3 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer3"`
	Timer4 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer4"`
	Timer5 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer5"`
	Timer6 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer6"`
	Timer7 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer7"`
	Timer8 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer8"`
	Timer9 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer9"`
	Timer10 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer10"`
	Timer11 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer11"`
	Timer12 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer12"`
	Timer13 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer13"`
	Timer14 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer14"`
	Timer15 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer15"`
	Timer16 struct {
		Enable int    `json:"Enable"`
		Mode   int    `json:"Mode"`
		Time   string `json:"Time"`
		Window int    `json:"Window"`
		Days   string `json:"Days"`
		Repeat int    `json:"Repeat"`
		Output int    `json:"Output"`
		Action int    `json:"Action"`
	} `json:"Timer16"`
}

func init() {

	homeDirName, err := os.UserHomeDir()
	checkErr(err)

	flag.String("cmd", "", "Command")
	flag.String("config", homeDirName+"/.tasproxy", "Configuration file: /path/to/file.yaml, default = "+homeDirName+"/.tasproxy")
	flag.String("custom", "", "Custom command")
	flag.String("device", "", "Device")
	flag.Bool("displayconfig", false, "Display configuration")
	flag.Bool("help", false, "Help")
	flag.String("ip", "", "IP")
	flag.Bool("json", false, "Output JSON")
	flag.Bool("list", false, "List Devices")
	flag.String("payload", "", "Timer Payload")
	flag.Int("timer", 0, "Timer Number")
	flag.Bool("timers", false, "List all timers")
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

	//
	//if !viper.IsSet("cmd") {
	//	fmt.Println("cmd not set")
	//	os.Exit(1)
	//}

	if (viper.IsSet("custom")) && (viper.IsSet("cmd")) {
		fmt.Println("custom or cmd cannot be used at the same time")
		os.Exit(1)
	}

	if (!viper.IsSet("custom")) && (!viper.IsSet("cmd")) {
		fmt.Println("either custom or cmd must be set")
		os.Exit(1)
	}

	// command to actually send to tasmota
	var sendCommand string

	// prep custom command to send
	if viper.IsSet("custom") {
		fmt.Println("custom set")

		// Timer4  {"Enable":1,         "Time":"16:23","Window":15,"Days":"SM00TF0","Repeat":0,"Output":2,"Action":2}
		// Timer16 {"Enable":0,"Mode":0,"Time":"13:11","Window":15,"Days":"1111111","Repeat":1,"Output":1,"Action":0}
		//junk := "Timer16 {\"Enable\":0,\"Mode\":0,\"Time\":\"13:11\",\"Window\":15,\"Days\":\"1111111\",\"Repeat\":1,\"Output\":1,\"Action\":0}"
		//fmt.Printf("Timer%d Payload: %s\n", viper.GetInt("timer"), viper.GetString("payload"))
		//fmt.Printf("junk: %s\n", junk)
		junk := viper.GetString("custom")
		fmt.Printf("        junk: %s\n", junk)
		fmt.Printf("junk escaped: %s\n", url.QueryEscape(junk))
		sendCommand = url.QueryEscape(junk)
	}

	// prep cmd to send
	if viper.IsSet("cmd") {
		fmt.Println("cmd set")

		// check if command is valid
		if isCommandValid(viper.GetString("cmd")) {
			// convert shorthand cmd to actual tasmota command
			sendCommand = commandList[viper.GetString("cmd")]
		} else {
			fmt.Printf("Command \"%s\" is invalid\n", viper.GetString("cmd"))
			os.Exit(1)
		}
	}

	// check if device is valid
	if checkDeviceValid(viper.GetString("device")) {
		if verbose {
			fmt.Printf("Device: %s found\n", viper.GetString("device"))
		}
	} else {
		fmt.Printf("Device: %s not found\n", viper.GetString("device"))
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("sendcommand: %s\n", sendCommand)
	}

	// send the command
	ipofdevice := viper.GetStringMap("devices")[viper.GetString("device")].(string)

	response, success := sendTasmota(ipofdevice, sendCommand)

	if !success {
		// could not send to tasmota
		fmt.Println("oops not successful")
		os.Exit(1)
	} else {

		if verbose {
			fmt.Printf("successful response: %s\n", prettyPrint(response))
		}

		if viper.IsSet("custom") {
			fmt.Println("run custom stuff here")
			var prettyJSON bytes.Buffer
			error := json.Indent(&prettyJSON, response, "", "\t")
			checkErr(error)
			fmt.Println(string(prettyJSON.Bytes()))
		}

		if viper.IsSet("cmd") {

			if verbose {
				fmt.Println("cmd is set")
			}

			cleanCommand := strings.ToLower(viper.GetString("cmd"))

			fmt.Printf("cleancommand before equalfold: %s\n", cleanCommand)

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

				// if status
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

					fmt.Printf("%s\n", powerState)

				}

				// if status all
				if strings.EqualFold(cleanCommand, "statusall") {
					fmt.Printf("%s\n", prettyPrint(res))
				}
			}

			// if timers
			if strings.EqualFold(cleanCommand, "timers") {
				res := AllTimers{}
				err := json.Unmarshal(response, &res)
				checkErr(err)

				if viper.GetBool("json") {
					// if wanting json output
					fmt.Printf("%s\n", prettyPrint(res))
				} else {
					// if wanting console output
					w := new(tabwriter.Writer)

					const padding = 1
					w.Init(os.Stdout, 0, 2, padding, ' ', 0)
					defer w.Flush()

					// forgive me, this is gross
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Name", "Enabled", "Mode", "Time", "Window", "Days", "Repeat", "Output", "Action")
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "-------", "-------", "----", "-----", "------", "-------", "------", "------", "------")
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer1", res.Timer1.Enable, res.Timer1.Mode, res.Timer1.Time, res.Timer1.Window, res.Timer1.Days, res.Timer1.Repeat, res.Timer1.Output, res.Timer1.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer2", res.Timer2.Enable, res.Timer2.Mode, res.Timer2.Time, res.Timer2.Window, res.Timer2.Days, res.Timer2.Repeat, res.Timer2.Output, res.Timer2.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer3", res.Timer3.Enable, res.Timer3.Mode, res.Timer3.Time, res.Timer3.Window, res.Timer3.Days, res.Timer3.Repeat, res.Timer3.Output, res.Timer3.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer4", res.Timer4.Enable, res.Timer4.Mode, res.Timer4.Time, res.Timer4.Window, res.Timer4.Days, res.Timer4.Repeat, res.Timer4.Output, res.Timer4.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer5", res.Timer5.Enable, res.Timer5.Mode, res.Timer5.Time, res.Timer5.Window, res.Timer5.Days, res.Timer5.Repeat, res.Timer5.Output, res.Timer5.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer5", res.Timer5.Enable, res.Timer5.Mode, res.Timer5.Time, res.Timer5.Window, res.Timer5.Days, res.Timer5.Repeat, res.Timer5.Output, res.Timer5.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer6", res.Timer6.Enable, res.Timer6.Mode, res.Timer6.Time, res.Timer6.Window, res.Timer6.Days, res.Timer6.Repeat, res.Timer6.Output, res.Timer6.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer7", res.Timer7.Enable, res.Timer7.Mode, res.Timer7.Time, res.Timer7.Window, res.Timer7.Days, res.Timer7.Repeat, res.Timer7.Output, res.Timer7.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer8", res.Timer8.Enable, res.Timer8.Mode, res.Timer8.Time, res.Timer8.Window, res.Timer8.Days, res.Timer8.Repeat, res.Timer8.Output, res.Timer8.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer9", res.Timer9.Enable, res.Timer9.Mode, res.Timer9.Time, res.Timer9.Window, res.Timer9.Days, res.Timer9.Repeat, res.Timer9.Output, res.Timer9.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer10", res.Timer10.Enable, res.Timer10.Mode, res.Timer10.Time, res.Timer10.Window, res.Timer10.Days, res.Timer10.Repeat, res.Timer10.Output, res.Timer10.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer11", res.Timer11.Enable, res.Timer11.Mode, res.Timer11.Time, res.Timer11.Window, res.Timer11.Days, res.Timer11.Repeat, res.Timer11.Output, res.Timer11.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer12", res.Timer12.Enable, res.Timer12.Mode, res.Timer12.Time, res.Timer12.Window, res.Timer12.Days, res.Timer12.Repeat, res.Timer12.Output, res.Timer12.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer13", res.Timer13.Enable, res.Timer13.Mode, res.Timer13.Time, res.Timer13.Window, res.Timer13.Days, res.Timer13.Repeat, res.Timer13.Output, res.Timer13.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer14", res.Timer14.Enable, res.Timer14.Mode, res.Timer14.Time, res.Timer14.Window, res.Timer14.Days, res.Timer14.Repeat, res.Timer14.Output, res.Timer14.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer15", res.Timer15.Enable, res.Timer15.Mode, res.Timer15.Time, res.Timer15.Window, res.Timer15.Days, res.Timer15.Repeat, res.Timer15.Output, res.Timer15.Action)
					fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer16", res.Timer16.Enable, res.Timer16.Mode, res.Timer16.Time, res.Timer16.Window, res.Timer16.Days, res.Timer16.Repeat, res.Timer16.Output, res.Timer16.Action)

					fmt.Println("\nFurther details available here: https://tasmota.github.io/docs/Timers/#json-payload-anatomy")
				}
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
		if verbose {
			fmt.Println("http status = ok")
		}
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
