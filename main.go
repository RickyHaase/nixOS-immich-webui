package main

import (
	"embed"
	"encoding/json"
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

const nixDir string = "test/nixos/"           //to actually modify the nix config used by the system, this const needs to be set for "/etc/nixos/"
const immichDir string = "/root/immich-app/"  //not certain where this will be in final prod but for now it's /root/immich-app
const tankImmich string = "test/tank/immich/" //really only for immich-config.json. Not certain where this will end up in the end

//go:embed internal/templates
var templates embed.FS

type NixConfig struct { // This struct MUST contain all NixOS config settings that will be modifiable via this interface
	TimeZone     string
	AutoUpgrade  bool   //also applies to allowReboot
	UpgradeTime  string //start of 1-hour window, interruption should be minimal during that window
	UpgradeLower string //value derived from UpgradeTime+30min
	UpgradeUpper string //value derived from UpgradeTime+60min
	Tailscale    bool
	TSAuthkey    string
	Email        string
	EmailPass    bool
}

type ImmichConfig struct {
	Backup          Backup          `json:"backup"`
	Notifications   Notifications   `json:"notifications"`
	Server          Server          `json:"server"`
	StorageTemplate StorageTemplate `json:"storageTemplate"`
}

type Backup struct {
	Database Database `json:"database"`
}

type Database struct {
	CronExpression string `json:"cronExpression"`
	Enabled        bool   `json:"enabled"`
	KeepLastAmount int    `json:"keepLastAmount"`
}

type Notifications struct {
	SMTP SMTP `json:"smtp"`
}

type SMTP struct {
	Enabled   bool      `json:"enabled"`
	From      string    `json:"from"`
	ReplyTo   string    `json:"replyTo"`
	Transport Transport `json:"transport"`
}

type Transport struct {
	Host       string `json:"host"`
	IgnoreCert bool   `json:"ignoreCert"`
	Password   string `json:"password"`
	Port       int16  `json:"port"`
	Username   string `json:"username"`
}

type Server struct {
	ExternalDomain   string `json:"externalDomain"`
	LoginPageMessage string `json:"loginPageMessage"`
	PublicUsers      bool   `json:"publicUsers"`
}

type StorageTemplate struct {
	Enabled                 bool   `json:"enabled"`
	HashVerificationEnabled bool   `json:"hashVerificationEnabled"`
	Template                string `json:"template"`
}

// // Need to set defaults and have a way to revert to them (eventually)
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
	tmpl, err := texttemplate.ParseFS(templates, "internal/templates/nixos/configuration.nix")
	if err != nil {
		fmt.Println("Error rendering template:", err)
		return err
	}

	outFile, err := os.Create(nixDir + "configuration.tmp")
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
	file, err := os.ReadFile(nixDir + "configuration.nix")
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

	// Parse settings out of immich-config.json
	immich, err := getImmichConfig()
	if err != nil {
		fmt.Println("Error parsing Immich Config", err)
		return nil, err
	}

	config.Email = immich.Notifications.SMTP.Transport.Username

	if immich.Notifications.SMTP.Transport.Password != "" {
		// fmt.Println("IF was met")
		config.EmailPass = true
	} else {
		// fmt.Println("ELSE was met")
		config.EmailPass = false
	}

	// fmt.Print(config.EmailPass)
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
	configPath := nixDir + "configuration.nix"
	backupPath := nixDir + "configuration.old"
	tmpPath := nixDir + "configuration.tmp"

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

func getStatus() string {
	cmd := exec.Command("systemctl", "show", "-p", "ActiveState", "--value", "immich-app.service")
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error getting status:", err)
		return "Error getting status"
	}

	status := string(output)
	switch status {
	case "active\n":
		return "Running"
	case "inactive\n":
		return "Stopped"
	default:
		return "Error getting status"
	}
}

func immichService(command string) error {
	cmd := exec.Command("systemctl", command, "immich-app.service")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error running %s against immich-app.service: %v\n", command, err)
		return err
	}
	return nil
}

func updateImmichContainer() error {
	path := immichDir + "docker-compose.yml"
	cmd := exec.Command("docker", "compose", "-f", path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to pull new containers: %w", err)
	}

	fmt.Println("compose pull completed successfully!")
	return nil
}

func getImmichConfig() (*ImmichConfig, error) { // Really no idea if this one is right. Seems like a lot happening
	file, err := os.Open(tankImmich + "immich-config.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var immichConfig ImmichConfig

	json.Unmarshal(byteValue, &immichConfig)

	return &immichConfig, nil
}

// Lots needs to change about this - atleast error reporting
func setImmichConfig(email string, password string) {
	// NOT using templating becuase we've got all the JSON we need... should cut down on errors but we need a "default" value somewhere
	immichConfig, err := getImmichConfig()
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	immichConfig.Notifications.SMTP.From = "Immich Server <" + email + ">"
	immichConfig.Notifications.SMTP.Transport.Username = email
	immichConfig.Notifications.SMTP.Transport.Password = password

	if email == "" || password == "" {
		immichConfig.Notifications.SMTP.Enabled = false
	} else {
		immichConfig.Notifications.SMTP.Enabled = true
	}

	b, err := json.MarshalIndent(immichConfig, "", "  ")
	if err != nil {
		fmt.Println("Error generating JSON:", err)
		return
	}

	// fmt.Println(string(b))

	fileName := tankImmich + "immich-config.tmp"

	if err := os.WriteFile(fileName, b, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	configFile := tankImmich + "immich-config.json"

	CopyFile(fileName, configFile)
}

// ====== Next Up ======
// WIP - Gmail notification configuration - Immich JSON & Immich Compose
// Function to start/stop tailscale
// Function to sign out of tailscale
// Function to use tailscale serve for immich
// Caddy basic Auth

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

	tmpl, err := htmltemplate.ParseFS(templates, "internal/templates/web/index.html")
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

	tmpl, err := htmltemplate.ParseFS(templates, "internal/templates/web/save.html")
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

func handleStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Status Request")
	w.Write([]byte(getStatus()))
}

func handleStop(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Stop Request")

	err := immichService("stop")
	if err != nil {
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte(getStatus()))
}

func handleStart(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Start Request")

	err := immichService("start")
	if err != nil {
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte(getStatus()))
}

func handleUpdate(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Recieved Update Request")

	if err := immichService("stop"); err != nil {
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}

	if err := updateImmichContainer(); err != nil {
		fmt.Println("Error updating Immich:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := immichService("start"); err != nil {
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte("Pulled new containers successfully"))

}

// func handleEmailGet(
// 	w http.ResponseWriter,
// 	r *http.Request,
// ) {
// 	fmt.Println("Email Get")

// }

func handleEmailPost(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Println("Email Post")

	// Do I need to validate submitted values in some way?
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}
	setImmichConfig(r.FormValue("gmail-address"), r.FormValue("gmail-password"))

	// Read new config and return HTML with new config injected
	//
	config := NixConfig{}

	// Parse settings out of immich-config.json - might need to refactor these 15 lines out into a function to maintan DRY best practice
	// (Also see repeated code in the "loadCurrentConfig()" function)
	immich, err := getImmichConfig()
	if err != nil {
		fmt.Println("Error parsing Immich Config", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config.Email = immich.Notifications.SMTP.Transport.Username
	if immich.Notifications.SMTP.Transport.Password != "" {
		// fmt.Println("IF was met")
		config.EmailPass = true
	} else {
		// fmt.Println("ELSE was met")
		config.EmailPass = false
	}

	htmlStr := `    <form id="email-form" action="/email" method="post">
        <label for="gmail-address">Gmail Address:</label>
        <input type="email" id="gmail-address" name="gmail-address" placeholder="example@gmail.com" value="{{if .Email}}{{.Email}}{{else}}{{end}}">        <label for="gmail-password">Gmail App Password:</label>
        <input type="password" id="gmail-password" name="gmail-password" placeholder="{{if .EmailPass}}password is set{{else}}fded beid aibr kxps{{end}}">
        <button type="submit" hx-post="/email" hx-target="#email-form">Submit</button>
        <br><small>Use your gmail account with an <a href="https://support.google.com/mail/answer/185833">app password</a> to allow for immich to send emails</small>
    </form>`
	// // need to store and parse return before writing it
	tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
	tmpl.Execute(w, config)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRoot)
	mux.HandleFunc("POST /save", handleSave)
	mux.HandleFunc("POST /apply", handleApply)
	mux.HandleFunc("GET /status", handleStatus)
	mux.HandleFunc("POST /stop", handleStop)
	mux.HandleFunc("POST /start", handleStart)
	mux.HandleFunc("POST /update", handleUpdate)
	// mux.HandleFunc("GET /email", handleEmailGet)
	mux.HandleFunc("POST /email", handleEmailPost)

	fmt.Println("Server started at http://localhost:8000")
	http.ListenAndServe("localhost:8000", mux)
}
