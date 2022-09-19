package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const applicationName string = "tasmota-proxy"
const applicationVersion = "v0.0.1"
const applicationUrl string = "https://github.com/smford/tasmota-proxy"

var (
	apikey      string
	interval    string
	message     string
	name        string
	silent      bool
	verbose     bool
	homeDirName string

	commandList = map[string]string{
		"on":     `Power%20On`,
		"off":    `Power%20Off`,
		"status": `Status0`,
	}
)

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
	verbose = true
	myCommand := viper.GetString("cmd")

	// check if device is valid
	if checkDeviceValid(viper.GetString("device")) {
		fmt.Printf("Device: %s found\n", viper.GetString("device"))
	} else {
		fmt.Printf("Device: %s not found\n", viper.GetString("device"))
		os.Exit(1)
	}

	if !isCommandValid(myCommand) {
		fmt.Printf("Command \"%s\" is invalid\n", myCommand)
		os.Exit(1)
	}

	// send the command
	ipofdevice := viper.GetStringMap("devices")[viper.GetString("device")].(string)
	sendTasmota(ipofdevice, commandList[myCommand])
}

func sendTasmota(tasmota string, cmd string) {
	client := &http.Client{}
	client.Timeout = time.Second * 15
	tasmota = url.QueryEscape(tasmota)
	if verbose {
		fmt.Printf("http://%s/cm?cmnd=%s\n", tasmota, cmd)
	}
	uri := fmt.Sprintf("http://%s/cm?cmnd=%s", tasmota, cmd)
	data := url.Values{
		"m": []string{message},
	}
	resp, err := client.PostForm(uri, data)
	if err != nil {
		log.Fatalf("client.PosFormt() failed with '%s'\n", err)
	}
	defer resp.Body.Close()

	tasmotaresponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("ioutil.ReadAll() failed with '%s'\n", err)
	}

	if verbose {
		fmt.Println("Response Code:", resp.StatusCode, "Response Text:", http.StatusText(resp.StatusCode), "Message:", tasmotaresponse)
	}

	if !silent {
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			fmt.Println("Success")
		}
	}
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
