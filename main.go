package main

import (
	"fmt"
	htmltemplate "html/template"

	// "io"
	"net/http"
	"os"

	// "os/exec"
	"regexp"
	// texttemplate "text/template"
)

const workingDir string = "test/" //to actually modify the nix config, this const needs to be set for "/etc/nixos/"

type NixConfig struct {
	TimeZone    string
	AutoUpgrade bool   //also applies to allowReboot
	UpgradeTime string //start of 3-hour window, interruption should be minimal during that window
	Tailscale   bool
	TSAuthkey   string
}

// Need to set defaults and have a way to revert to them (eventually)
// defaults := NixConfig{
// 	TimeZone: "America/New_York",
// 	AutoUpgrade: true,
// 	UpgradeTime: "2:00",
// 	Tailscale: false,
// 	TSAuthkey: "",
// }

// Helper function to parse boolean values from the configuration file - thanks ChatGPT :)
func parseBooleanSetting(fileContent []byte, setting string) (bool, error) {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*(true|false)\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		return false, fmt.Errorf("%s not found", setting)
	}
	return string(match[1]) == "true", nil
}

func parseStringSetting(fileContent []byte, setting string) (string, error) {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*"(.*?)"\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		return "", fmt.Errorf("%s not found", setting)
	}
	return string(match[1]), nil
}

func parseAuthKeySetting(fileContent []byte) (string, error) {
	re := regexp.MustCompile(`\btskey-auth-[a-zA-Z0-9]+-[a-zA-Z0-9]+\b`)
	match := re.Find(fileContent)
	if match == nil {
		return "", fmt.Errorf("tskey-auth not found")
	}
	return string(match), nil
}

// return error and handle in page render function... see wiki project. perhaps upon receiving error, it does not render the webpage but instead says "oops, something went wrong :/"
func loadCurrentConfig() (*NixConfig, error) {
	file, err := os.ReadFile(workingDir + "configuration.nix")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}

	config := NixConfig{}

	// Parse the relevant values out of the settings in the config file
	config.TimeZone, err = parseStringSetting(file, "time.timeZone")
	if err != nil {
		fmt.Println("Error parsing TimeZone:", err)
		return nil, err
	}

	config.AutoUpgrade, err = parseBooleanSetting(file, "system.autoUpgrade.enable")
	if err != nil {
		fmt.Println("Error parsing AutoUpgrade Enable:", err)
		return nil, err
	}

	config.UpgradeTime, err = parseStringSetting(file, "system.autoUpgrade.dates")
	if err != nil {
		fmt.Println("Error parsing UpdgradeTime:", err)
		return nil, err
	}

	config.Tailscale, err = parseBooleanSetting(file, "services.tailscale.enable")
	if err != nil {
		fmt.Println("Error parsing Tailscale Enable", err)
		return nil, err
	}

	config.TSAuthkey, err = parseAuthKeySetting(file)
	if err != nil {
		fmt.Println("Error parsing Tailscale AuthKey", err)
		return nil, err
	}

	// Print parsed values
	// fmt.Printf("time.timeZone is set to %s\n", config.TimeZone)
	// fmt.Printf("system.autoUpgrade.enable is set to %t\n", config.AutoUpgrade)
	// fmt.Printf("system.autoUpgrade.dates is set to %s\n", config.UpgradeTime)
	// fmt.Printf("services.tailscale.enable is set to %t\n", config.Tailscale)
	// fmt.Printf("tskey-auth is set to %s\n", config.TSAuthkey)

	return &config, nil
}

// Function to render settings page with current configuration state
//
// Function to write temp config
//
// Function to apply temp config
//
// ====== Next Up ======
// Function to start/stop tailscale
// Function to sign out of tailscale
// Function to start/stop immich
// Function to update immich
//
// Caddy basic Auth
// Gmail notification configuration - Immich JSON & Immich Compose

func handleRoot(
	w http.ResponseWriter,
	r *http.Request,
) {
	config, err := loadCurrentConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFiles("internal/templates/web/index.html")
	if err != nil {
		fmt.Println("Error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, config)
	// renderTemplate(w, "view", config)

	// loadCurrentConfig()
	// http.ServeFile(w, r, "internal/templates/web/index.html")
}

func handleSave(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Save")
}

// APPLY will eventually become much more complicated with a modal pop-up, error checking, rollback, etc. enbaled (presumably) by HTMX
func handleApply(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Apply")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRoot)
	mux.HandleFunc("POST /save", handleSave)
	mux.HandleFunc("POST /apply", handleApply)

	fmt.Println("Server started at http://localhost:8000")
	http.ListenAndServe("localhost:8000", mux)
}
