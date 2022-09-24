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

const applicationName string = "tasmota-cli"
const applicationVersion = "v0.2"
const applicationUrl string = "https://github.com/smford/tasmota-cli"

var (
	verbose     bool
	homeDirName string
	ipofdevice  string

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

// structure of all timers, super gross
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

	flag.String("cmd", "", "Command: on, off, status, statusall, timers")
	flag.String("config", homeDirName+"/.tascli", "Configuration file: /path/to/file.yaml, default = "+homeDirName+"/.tascli")
	flag.String("custom", "", "Custom escaped command string to send")
	flag.String("device", "", "Device")
	flag.Bool("displayconfig", false, "Display configuration")
	flag.Bool("help", false, "Help")
	flag.String("host", "", "IP address or hostname of a device")
	flag.Bool("json", false, "Output JSON")
	flag.Bool("list", false, "List Devices")
	flag.Bool("version", false, "Version")

	// temp
	flag.Bool("verbose", false, "Be verbose")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err = viper.BindPFlags(pflag.CommandLine)

	checkErr(err)

	viper.SetEnvPrefix("TASCLI")
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

	// prevent conflicting arguments from breaking logic
	if (viper.IsSet("custom")) && (viper.IsSet("cmd")) {
		fmt.Println("--custom or --cmd cannot be used at the same time")
		os.Exit(1)
	}

	if (!viper.IsSet("custom")) && (!viper.IsSet("cmd")) {
		fmt.Println("either --custom or --cmd must be set")
		os.Exit(1)
	}

	if (viper.IsSet("device")) && (viper.IsSet("host")) {
		fmt.Println("--device and --host cannot be used at the same time")
		os.Exit(1)
	}

	if (!viper.IsSet("device")) && (!viper.IsSet("host")) {
		fmt.Println("either --device or --host must be set")
		os.Exit(1)
	}

	// command to actually send to tasmota
	var sendCommand string

	// prep custom command to send
	if viper.IsSet("custom") {
		if verbose {
			fmt.Printf("Custom Command: %s\n", viper.GetString("custom"))
			fmt.Printf("Custom Command Escaped: %s\n", url.QueryEscape(viper.GetString("custom")))
		}
		sendCommand = url.QueryEscape(viper.GetString("custom"))
	}

	// prep cmd to send
	if viper.IsSet("cmd") {
		// check if command is valid
		if isCommandValid(viper.GetString("cmd")) {
			// convert shorthand cmd to actual tasmota compatable command
			sendCommand = commandList[viper.GetString("cmd")]
		} else {
			fmt.Printf("Command \"%s\" is invalid\n", viper.GetString("cmd"))
			os.Exit(1)
		}
	}

	if viper.IsSet("host") {
		ipofdevice = viper.GetString("host")
	} else {
		// check if device is valid
		if checkDeviceValid(viper.GetString("device")) {
			if verbose {
				fmt.Printf("Device: %s found\n", viper.GetString("device"))
			}

			ipofdevice = viper.GetStringMap("devices")[viper.GetString("device")].(string)

		} else {
			fmt.Printf("Device: %s not found\n", viper.GetString("device"))
			os.Exit(1)
		}
	}

	// send the command to tasmota
	response, success := sendTasmota(ipofdevice, sendCommand)

	if !success {
		// could not send to tasmota
		fmt.Println("Error: Could not connect to device")
		os.Exit(1)
	} else {

		if verbose {
			fmt.Printf("Successful Response: %s\n", prettyPrint(response))
		}

		// if custom command was sent
		if viper.IsSet("custom") {
			// as response will be in an unknown json format, just make pretty indents and dump to console
			var niceJSON bytes.Buffer
			error := json.Indent(&niceJSON, response, "", "\t")
			checkErr(error)
			fmt.Println(string(niceJSON.Bytes()))
			os.Exit(0)
		}

		// if baked in cmd was sent
		if viper.IsSet("cmd") {

			cleanCommand := strings.ToLower(viper.GetString("cmd"))

			// start: if power on or power off
			if strings.EqualFold(cleanCommand, "on") || strings.EqualFold(cleanCommand, "off") {
				res := PowerResponse{}
				err := json.Unmarshal(response, &res)
				checkErr(err)
				fmt.Printf("%s:%s\n", viper.GetString("device"), res.Power)
				os.Exit(0)
			}
			// end: if power on or power off

			// start: if status or statusall
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
					os.Exit(0)
				}

				// if statusall
				if strings.EqualFold(cleanCommand, "statusall") {
					fmt.Printf("%s\n", prettyPrint(res))
					os.Exit(0)
				}
			}
			// end: if status or statusall

			// start: if timers
			if strings.EqualFold(cleanCommand, "timers") {
				res := AllTimers{}
				err := json.Unmarshal(response, &res)
				checkErr(err)

				// if wanting json output
				if viper.GetBool("json") {
					fmt.Printf("%s\n", prettyPrint(res))
					os.Exit(0)
				} else {

					// if wanting console output
					printTimers(res)
					os.Exit(0)
				}
				// end: if timers

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

	// fix
	//if err != nil {
	//	log.Fatal("NewRequest: ", err)
	//}
	checkErr(err)

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
      --cmd [x]             Commands: on, off, status, statusall, timers
      --config [file]       Configuration file: /path/to/file.yaml, default = "` + homeDirName + `"/.tascli"
      --custom [command]    Custom escaped command string to send
      --device [name]       Name of device
      --displayconfig       Display configuration
      --help                Display help
      --host [address]      IP address or hostname of device
      --json                Output JSON
      --list                List all configured devices
      --verbose             Be verbose
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

	w := new(tabwriter.Writer)

	const padding = 1
	w.Init(os.Stdout, 0, 2, padding, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\n", "Config", "Setting")
	fmt.Fprintf(w, "%s\t%s\n", "------", "-------")

	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%v\n", k, allmysettings[k])
	}
}

// list devices
func displayDevices() {
	if viper.IsSet("devices") {
		// do sorting
		var keys []string
		for k := range viper.GetStringMap("devices") {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// print the list
		w := new(tabwriter.Writer)

		const padding = 1
		w.Init(os.Stdout, 0, 2, padding, ' ', 0)
		defer w.Flush()

		fmt.Fprintf(w, "%s\t%s\n", "IP", "Name")
		fmt.Fprintf(w, "%s\t%s\n", "--", "----")

		for _, k := range keys {
			fmt.Fprintf(w, "%s\t%s\n", viper.GetStringMap("devices")[k].(string), k)
		}
		fmt.Println("======")

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

// print all timers
func printTimers(myTimers AllTimers) {
	w := new(tabwriter.Writer)

	const padding = 1
	w.Init(os.Stdout, 0, 2, padding, ' ', 0)
	defer w.Flush()

	// forgive me, this is gross
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "Name", "Enabled", "Mode", "Time", "Window", "Days", "Repeat", "Output", "Action")
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "-------", "-------", "----", "-----", "------", "-------", "------", "------", "------")
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer1", myTimers.Timer1.Enable, myTimers.Timer1.Mode, myTimers.Timer1.Time, myTimers.Timer1.Window, myTimers.Timer1.Days, myTimers.Timer1.Repeat, myTimers.Timer1.Output, myTimers.Timer1.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer2", myTimers.Timer2.Enable, myTimers.Timer2.Mode, myTimers.Timer2.Time, myTimers.Timer2.Window, myTimers.Timer2.Days, myTimers.Timer2.Repeat, myTimers.Timer2.Output, myTimers.Timer2.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer3", myTimers.Timer3.Enable, myTimers.Timer3.Mode, myTimers.Timer3.Time, myTimers.Timer3.Window, myTimers.Timer3.Days, myTimers.Timer3.Repeat, myTimers.Timer3.Output, myTimers.Timer3.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer4", myTimers.Timer4.Enable, myTimers.Timer4.Mode, myTimers.Timer4.Time, myTimers.Timer4.Window, myTimers.Timer4.Days, myTimers.Timer4.Repeat, myTimers.Timer4.Output, myTimers.Timer4.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer5", myTimers.Timer5.Enable, myTimers.Timer5.Mode, myTimers.Timer5.Time, myTimers.Timer5.Window, myTimers.Timer5.Days, myTimers.Timer5.Repeat, myTimers.Timer5.Output, myTimers.Timer5.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer5", myTimers.Timer5.Enable, myTimers.Timer5.Mode, myTimers.Timer5.Time, myTimers.Timer5.Window, myTimers.Timer5.Days, myTimers.Timer5.Repeat, myTimers.Timer5.Output, myTimers.Timer5.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer6", myTimers.Timer6.Enable, myTimers.Timer6.Mode, myTimers.Timer6.Time, myTimers.Timer6.Window, myTimers.Timer6.Days, myTimers.Timer6.Repeat, myTimers.Timer6.Output, myTimers.Timer6.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer7", myTimers.Timer7.Enable, myTimers.Timer7.Mode, myTimers.Timer7.Time, myTimers.Timer7.Window, myTimers.Timer7.Days, myTimers.Timer7.Repeat, myTimers.Timer7.Output, myTimers.Timer7.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer8", myTimers.Timer8.Enable, myTimers.Timer8.Mode, myTimers.Timer8.Time, myTimers.Timer8.Window, myTimers.Timer8.Days, myTimers.Timer8.Repeat, myTimers.Timer8.Output, myTimers.Timer8.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer9", myTimers.Timer9.Enable, myTimers.Timer9.Mode, myTimers.Timer9.Time, myTimers.Timer9.Window, myTimers.Timer9.Days, myTimers.Timer9.Repeat, myTimers.Timer9.Output, myTimers.Timer9.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer10", myTimers.Timer10.Enable, myTimers.Timer10.Mode, myTimers.Timer10.Time, myTimers.Timer10.Window, myTimers.Timer10.Days, myTimers.Timer10.Repeat, myTimers.Timer10.Output, myTimers.Timer10.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer11", myTimers.Timer11.Enable, myTimers.Timer11.Mode, myTimers.Timer11.Time, myTimers.Timer11.Window, myTimers.Timer11.Days, myTimers.Timer11.Repeat, myTimers.Timer11.Output, myTimers.Timer11.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer12", myTimers.Timer12.Enable, myTimers.Timer12.Mode, myTimers.Timer12.Time, myTimers.Timer12.Window, myTimers.Timer12.Days, myTimers.Timer12.Repeat, myTimers.Timer12.Output, myTimers.Timer12.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer13", myTimers.Timer13.Enable, myTimers.Timer13.Mode, myTimers.Timer13.Time, myTimers.Timer13.Window, myTimers.Timer13.Days, myTimers.Timer13.Repeat, myTimers.Timer13.Output, myTimers.Timer13.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer14", myTimers.Timer14.Enable, myTimers.Timer14.Mode, myTimers.Timer14.Time, myTimers.Timer14.Window, myTimers.Timer14.Days, myTimers.Timer14.Repeat, myTimers.Timer14.Output, myTimers.Timer14.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer15", myTimers.Timer15.Enable, myTimers.Timer15.Mode, myTimers.Timer15.Time, myTimers.Timer15.Window, myTimers.Timer15.Days, myTimers.Timer15.Repeat, myTimers.Timer15.Output, myTimers.Timer15.Action)
	fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%d\n", "Timer16", myTimers.Timer16.Enable, myTimers.Timer16.Mode, myTimers.Timer16.Time, myTimers.Timer16.Window, myTimers.Timer16.Days, myTimers.Timer16.Repeat, myTimers.Timer16.Output, myTimers.Timer16.Action)

	fmt.Println("\nFurther details available here: https://tasmota.github.io/docs/Timers/#json-payload-anatomy\n")
}
