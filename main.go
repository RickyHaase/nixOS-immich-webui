package main

import (
	"embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	texttemplate "text/template"
	"time"
)

// Perhaps setup an init function that checks if binary is running in dev or prod to set these paths
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

// Need Default NixOS Config
// Need Default Immich Config
// Will need way to apply default configs. Will create default configs and use them when creating the "intial setup" flow
// Will change template parsing to happen at program intiialization rather than at runtime

// Helper function to parse boolean values from the configuration file - thanks ChatGPT :)
// Might revise the structure and error handling on these now that I've got a better understanding of how they work
func parseBooleanSetting(fileContent []byte, setting string) (bool, error) {
	slog.Debug("parseBooleanSetting", "setting", setting)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*(true|false)\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		slog.Debug("No Match Found", "setting", setting)
		return false, fmt.Errorf("%s not found", setting)
	}
	return string(match[1]) == "true", nil
}

func parseStringSetting(fileContent []byte, setting string) (string, error) {
	slog.Debug("parseStringSetting", "setting", setting)
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*"(.*?)"\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		slog.Debug("No Match Found", "setting", setting)
		return "", fmt.Errorf("%s not found", setting)
	}
	return string(match[1]), nil
}

func parseAuthKeySetting(fileContent []byte) (string, error) {
	slog.Debug("parseAuthKeySetting()")
	re := regexp.MustCompile(`\btskey-auth-[a-zA-Z0-9]+-[a-zA-Z0-9]+\b`)
	match := re.Find(fileContent)
	if match == nil {
		slog.Debug("No Match Found for authkey")
		return "", fmt.Errorf("tskey-auth not found")
	}
	return string(match), nil
}

func parseBool(value string) bool {
	slog.Debug("parseBool(string)", "string", value)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		slog.Error("| Error parsing boolean value - defaulting to False |", "err", err)
		return false
	}
	return boolValue
}

func saveTmpFile(config *NixConfig) error {
	slog.Debug("saveTmpFile()")
	tmpl, err := texttemplate.ParseFS(templates, "internal/templates/nixos/configuration.nix")
	if err != nil {
		slog.Debug("| Error rendering template |", "err", err)
		return err
	}

	outFile, err := os.Create(nixDir + "configuration.tmp")
	if err != nil {
		slog.Debug("| Error creating .tmp file |", "err", err)
		return err
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, config)
	if err != nil {
		slog.Debug("| Error writing .tmp file |", "err", err)
		return err
	}

	return nil
}

func getLowerUpper(timeStr string) (string, string, error) { // I like this because it inadvertently performs server-side validation of the time sent to the server
	slog.Debug("getLowerUpper()")
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		slog.Debug("Error parsing time:", "err", err)
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
	slog.Debug("loadCurrentConfig()")
	file, err := os.ReadFile(nixDir + "configuration.nix")
	if err != nil {
		slog.Debug("Error opening file:", "err", err)
		return nil, err
	}

	config := NixConfig{}

	// Parse the relevant values out of the settings in the config file
	config.TimeZone, err = parseStringSetting(file, "time.timeZone")
	if err != nil {
		slog.Debug("Error parsing TimeZone:", "err", err)
		return nil, err
	}

	config.AutoUpgrade, err = parseBooleanSetting(file, "system.autoUpgrade.enable")
	if err != nil {
		slog.Debug("Error parsing AutoUpgrade Enable:", "err", err)
		return nil, err
	}

	config.UpgradeTime, err = parseStringSetting(file, "system.autoUpgrade.dates")
	if err != nil {
		slog.Debug("Error parsing UpdgradeTime:", "err", err)
		return nil, err
	}

	config.Tailscale, err = parseBooleanSetting(file, "services.tailscale.enable")
	if err != nil {
		slog.Debug("Error parsing Tailscale Enable", "err", err)
		return nil, err
	}

	config.TSAuthkey, err = parseAuthKeySetting(file)
	if err != nil {
		slog.Debug("Error parsing Tailscale AuthKey", "err", err)
		return nil, err
	}

	// Parse settings out of immich-config.json
	immich, err := getImmichConfig()
	if err != nil {
		slog.Debug("Error parsing Immich Config", "err", err)
		return nil, err
	}

	config.Email = immich.Notifications.SMTP.Transport.Username

	if immich.Notifications.SMTP.Transport.Password != "" {
		slog.Debug("IF was met")
		config.EmailPass = true
	} else {
		slog.Debug("ELSE was met")
		config.EmailPass = false
	}
	slog.Debug("Password Boolean", "EmailPass", config.EmailPass)

	return &config, nil
}

func CopyFile(src, dst string) error {
	slog.Debug("CopyFile()")
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
	slog.Debug("switchConfig()")
	configPath := nixDir + "configuration.nix"
	backupPath := nixDir + "configuration.old"
	tmpPath := nixDir + "configuration.tmp"

	slog.Info("Backing up configuration.nix to configuration.old...")
	if err := CopyFile(configPath, backupPath); err != nil {
		slog.Debug("Error backing up config file", "err", err)
		return err
	}

	slog.Info("Replacing configuration.nix with configuration.tmp...")
	if err := CopyFile(tmpPath, configPath); err != nil {
		slog.Debug("Error replacing config file", "err", err)
		return err
	}

	slog.Info("Configuration file swtich complete.")
	return nil
}

func applyChanges() error {
	slog.Debug("applyChanges()")
	cmd := exec.Command("nixos-rebuild", "switch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Debug("| error running 'nixos-rebuild switch' |", "err", err)
		return fmt.Errorf("failed to execute nixos-rebuild: %w", err)
	}

	slog.Info("NixOS rebuild completed successfully.")
	return nil
}

func getStatus() string {
	slog.Debug("getStatus()")
	cmd := exec.Command("systemctl", "show", "-p", "ActiveState", "--value", "immich-app.service")
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		slog.Error("| Error getting status of immich-app.service |", "err", err)
		return "Error getting status"
	}

	status := string(output)
	switch status {
	case "active\n":
		return "Running"
	case "inactive\n":
		return "Stopped"
	default:
		slog.Error("| Unexpected status of immich-app.service |", "err", err)
		return "Error getting status"
	}
}

func immichService(command string) error {
	slog.Debug("immichService(string)", "string", command)
	cmd := exec.Command("systemctl", command, "immich-app.service")
	err := cmd.Run()
	if err != nil {
		slog.Error("Error running %s against immich-app.service: %v", command, err)
		return err
	}
	return nil
}

func updateImmichContainer() error {
	slog.Debug("updateImmichContainer()")
	path := immichDir + "docker-compose.yml"
	cmd := exec.Command("docker", "compose", "-f", path, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println(cmd)
	err := cmd.Run()
	if err != nil {
		slog.Debug("| Error executing 'docker compose pull' |", "cmd", cmd, "err", err)
		return fmt.Errorf("failed to pull new containers: %w", err)
	}

	slog.Info("compose pull completed successfully")
	return nil
}

func getImmichConfig() (*ImmichConfig, error) { // Really no idea if this one is right. Seems like a lot happening
	slog.Debug("getImmichConfig()")
	file, err := os.Open(tankImmich + "immich-config.json")
	if err != nil {
		slog.Debug("| Error opening immich config file |", "err", err)
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var immichConfig ImmichConfig

	json.Unmarshal(byteValue, &immichConfig)

	return &immichConfig, nil
}

func setImmichConfig(email string, password string) error {
	slog.Debug("setImmichConfig()")
	// NOT using templating becuase we've got all the JSON we need... should cut down on errors but we need a "default" value somewhere
	immichConfig, err := getImmichConfig()
	if err != nil {
		slog.Debug("Error reading immich config file", "err", err)
		return err
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
		slog.Debug("Error generating JSON", "err", err)
		return err
	}

	slog.Debug(string(b))

	fileName := tankImmich + "immich-config.tmp"

	if err := os.WriteFile(fileName, b, 0644); err != nil {
		slog.Debug("Error writing to file:", "err", err)
		return err
	}

	configFile := tankImmich + "immich-config.json"

	CopyFile(fileName, configFile)

	slog.Info("Immich config Set")

	return nil
}

func handleRoot(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("| Received Request at root |", "IP", r.Header.Get("X-Forwarded-For"))

	config, err := loadCurrentConfig()
	if err != nil {
		slog.Error("| Error loading config |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFS(templates, "internal/templates/web/index.html")
	if err != nil {
		slog.Error("| Error rendering template |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// I should probably add a check for error on execute to catch incase there are values the template requires that are missing
	tmpl.Execute(w, config)
}

func handleSave(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Save Request")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing form |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	slog.Debug("Received Form", "body", r.Form)

	config := &NixConfig{
		TimeZone:    r.FormValue("timezone"),
		AutoUpgrade: parseBool(r.FormValue("auto-updates")),
		UpgradeTime: r.FormValue("update-time"),
		Tailscale:   parseBool(r.FormValue("tailscale")),
		TSAuthkey:   r.FormValue("tailscale-authkey"),
	}

	t1, t2, err := getLowerUpper(config.UpgradeTime)
	if err != nil {
		slog.Error("| Error calculating time setting |", "err", err)
		http.Error(w, "Issue with time setting"+err.Error(), http.StatusInternalServerError)
		return
	}
	config.UpgradeLower = t1
	config.UpgradeUpper = t2

	slog.Debug("Updated config", "config", config)

	err = saveTmpFile(config)
	if err != nil {
		slog.Error("| Error saving tmp file |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := htmltemplate.ParseFS(templates, "internal/templates/web/save.html")
	if err != nil {
		slog.Error("| Error rendering save template |", "err", err)
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
	slog.Info("Received Apply Request")

	if err := switchConfig(); err != nil {
		slog.Error("| Error when switching config files |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := applyChanges(); err != nil {
		slog.Error("| Error Applying Changes |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// To-Do make a better confirmation page
	w.Write([]byte("Rebuild Completed Successfully"))
}

func handleStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	// fmt.Println("Received Status Request")
	slog.Debug("Received Status Request")
	w.Write([]byte(getStatus()))
}

func handleStop(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Stop Request")

	err := immichService("stop")
	if err != nil {
		slog.Error("| Error stopping immich-app.service |", "err", err)
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}

	// w.Write([]byte(getStatus()))
}

func handleStart(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Start Request")

	err := immichService("start")
	if err != nil {
		slog.Error("| Error starting immich-app.service |", "err", err)
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}

	// w.Write([]byte(getStatus()))
}

func handleUpdate(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Update Request")

	if err := immichService("stop"); err != nil {
		slog.Error("| Error stopping immich-app.service |", "err", err)
		http.Error(w, "Issue stopping Immich"+err.Error(), http.StatusInternalServerError)
	}

	if err := updateImmichContainer(); err != nil {
		slog.Error("| Error updating Immich |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := immichService("start"); err != nil {
		slog.Error("| Error starting immich-app.service |", "err", err)
		http.Error(w, "Issue starting Immich"+err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte("Pulled new containers successfully"))

}

// This function is a tad messy and ought to be cleaned up
func handleEmailPost(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Email Post")

	err := r.ParseForm()
	if err != nil {
		slog.Error("| Error parsing email form submission |", "err", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Do I need to validate submitted values in some way?
	if err := setImmichConfig(r.FormValue("gmail-address"), r.FormValue("gmail-password")); err != nil {
		slog.Error("| Failed to set Immich config |", "err", err)
		http.Error(w, "Failed to set Immich config.", http.StatusInternalServerError)
		return
	}

	// The rest of the function below this line should probably be tidied up and refactored
	config := NixConfig{}

	// Parse settings out of immich-config.json - might need to refactor these 15 lines out into a function to maintan DRY best practice
	// (Also see repeated code in the "loadCurrentConfig()" function)
	immich, err := getImmichConfig()
	if err != nil {
		slog.Error("| Error parisng immich-config.json |", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config.Email = immich.Notifications.SMTP.Transport.Username
	if immich.Notifications.SMTP.Transport.Password != "" {
		slog.Debug("Contains Password = True")
		config.EmailPass = true
	} else {
		slog.Debug("Contains Password = False")
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

func handlePoweroff(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Poweroff Request")

	cmd := exec.Command("poweroff")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("| Error executing poweroff |", "err", err)
		http.Error(w, "Failed to execute poweroff", http.StatusInternalServerError)
	}
}

func handleReboot(
	w http.ResponseWriter,
	r *http.Request,
) {
	slog.Info("Received Reboot Request")

	cmd := exec.Command("reboot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		slog.Error("| Error executing reboot |", "err", err)
		http.Error(w, "Failed to execute reboot", http.StatusInternalServerError)
	}
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
	mux.HandleFunc("POST /email", handleEmailPost)
	mux.HandleFunc("POST /poweroff", handlePoweroff)
	mux.HandleFunc("POST /reboot", handleReboot)

	// Probably need a 404/Error page that hyperlinks back to the main page

	// Need to make debug mode dynamic
	// slog.SetLogLoggerLevel(slog.LevelDebug)

	slog.Info("Server started at http://localhost:8000")
	http.ListenAndServe("localhost:8000", mux)
}
