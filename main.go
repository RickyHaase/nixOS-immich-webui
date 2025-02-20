package main

import (
	"fmt"
	htmltemplate "html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	texttemplate "text/template"
	"time"
)

const workingDir string = "test/" //to actually modify the nix config used by the system, this const needs to be set for "/etc/nixos/"

type NixConfig struct { // This struct MUST contain all NixOS config settings that will be modifiable via this interface
	TimeZone     string
	AutoUpgrade  bool   //also applies to allowReboot
	UpgradeTime  string //start of 1-hour window, interruption should be minimal during that window
	UpgradeLower string //value derived from UpgradeTime+30min
	UpgradeUpper string //value derived from UpgradeTime+60min
	Tailscale    bool
	TSAuthkey    string
}

// Need to set defaults and have a way to revert to them (eventually)
// defaults := &NixConfig{
// 	TimeZone: "America/New_York",
// 	AutoUpgrade: true,
// 	UpgradeTime: "2:00",
// 	Tailscale: false,
// 	TSAuthkey: "",
// }

// Helper function to parse boolean values from the configuration file - thanks ChatGPT :)
// Might revise the structure and error handling on these now that I've got a better understanding of how they work
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

func parseBool(value string) bool {
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		// Should probably actually do something with the error rather than just default to false
		return false
	}
	return boolValue
}

func saveTmpFile(config *NixConfig) error {
	tmpl, err := texttemplate.ParseFiles("internal/templates/nixos/configuration.nix")
	if err != nil {
		fmt.Println("Error rendering template:", err)
		return err
	}

	outFile, err := os.Create(workingDir + "configuration.tmp")
	if err != nil {
		fmt.Println("Error creating .tmp file:", err)
		return err
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, config)
	if err != nil {
		fmt.Println("Error writing .tmp file:", err)
		return err
	}

	return nil
}

func getLowerUpper(timeStr string) (string, string, error) { // I like this because it inadvertently performs server-side validation of the time sent to the server
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return "", "", err
	}

	t1 := t.Add(30 * time.Minute)
	t2 := t.Add(time.Hour)

	newTimeStr1 := t1.Format("15:04")
	newTimeStr2 := t2.Format("15:04")

	return newTimeStr1, newTimeStr2, nil
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

	return &config, nil
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", src, dst, err)
	}

	return nil
}

func switchConfig() error {
	configPath := workingDir + "configuration.nix"
	backupPath := workingDir + "configuration.old"
	tmpPath := workingDir + "configuration.tmp"

	fmt.Println("Backing up configuration.nix to configuration.old...")
	if err := CopyFile(configPath, backupPath); err != nil { //need to learn/understand this format of error-checking
		return err
	}

	fmt.Println("Replacing configuration.nix with configuration.tmp...")
	if err := CopyFile(tmpPath, configPath); err != nil { //need to learn/understand this format of error-checking
		return err
	}

	fmt.Println("Configuration update complete.")
	return nil
}

func applyChanges() error {
	cmd := exec.Command("nixos-rebuild", "switch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute nixos-rebuild: %w", err)
	}

	fmt.Println("NixOS rebuild completed successfully.")
	return nil
}

// ====== Next Up ======
// Function to start/stop immich
// Function to update immich
//
// Function to start/stop tailscale
// Function to sign out of tailscale
// Function to use tailscale serve for immich
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

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	config := &NixConfig{
		TimeZone:    r.FormValue("timezone"),
		AutoUpgrade: parseBool(r.FormValue("auto-updates")),
		UpgradeTime: r.FormValue("update-time"),
		Tailscale:   parseBool(r.FormValue("tailscale")),
		TSAuthkey:   r.FormValue("tailscale-authkey"),
	}

	// fmt.Printf("Received Body: %+v \n", config)

	t1, t2, err := getLowerUpper(config.UpgradeTime)
	if err != nil {
		http.Error(w, "Issue with time setting"+err.Error(), http.StatusInternalServerError)
		return
	}
	config.UpgradeLower = t1
	config.UpgradeUpper = t2

	// fmt.Printf("Updated config: %+v \n", config)

	err = saveTmpFile(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFiles("internal/templates/web/save.html")
	if err != nil {
		fmt.Println("Error rendering template:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, config)
}

// APPLY will eventually become much more complicated with a modal pop-up, error checking, rollback, etc. enbaled (presumably) by HTMX
func handleApply(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Apply")

	if err := switchConfig(); err != nil {
		fmt.Println("Error when switching config:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := applyChanges(); err != nil {
		fmt.Println("Error when rebuilding NixOS:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Rebuild Completed Successfully"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRoot)
	mux.HandleFunc("POST /save", handleSave)
	mux.HandleFunc("POST /apply", handleApply)

	fmt.Println("Server started at http://localhost:8000")
	http.ListenAndServe("localhost:8000", mux)
}
