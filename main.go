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
const applicationVersion = "v0.0.3"
const applicationUrl string = "https://github.com/smford/tasmota-proxy"

var (
	apikey      string
	verbose     bool
	homeDirName string

	commandList = map[string]string{
		"on":     `Power%20On`,
		"off":    `Power%20Off`,
		"status": `Status0`,
	}
)

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
	verbose = true

	if !viper.IsSet("cmd") {
		fmt.Println("Error, no command defined")
		os.Exit(1)
	}

	myCommand := viper.GetString("cmd")

	// check if device is valid
	if checkDeviceValid(viper.GetString("device")) {
		fmt.Printf("Device: %s found\n", viper.GetString("device"))
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
			fmt.Printf("Prettyprint:\n%s\n--\n", prettyPrint(res))
			fmt.Printf("State: %s\n", res.Power)
		}

		// if status

	}
}

// send a command to the tasmota
func sendTasmota(ip string, cmd string) ([]byte, bool) {

	url := fmt.Sprintf("http://%s/cm?cmnd=%s", ip, cmd)

	fmt.Printf("URL: %s\n", url)

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
