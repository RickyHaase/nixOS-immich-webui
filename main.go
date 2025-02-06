package main

import (
	"fmt"
	htmltemplate "html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	texttemplate "text/template"
)

const workingDir string = "test/" //to actually modify the nix config, this const needs to be set for "/etc/nixos/"

// Not sure how much of the initial nix config is necessary but I think I'll likely end up splitting things out into different imports to correspond with different settings pages in the UI
const nixosConfigTemplate = `
{ config, pkgs, ... }:

{
  imports = [ ./hardware-configuration.nix ];

  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  networking.hostName = "nixos";
  networking.networkmanager.enable = true;
  time.timeZone = "America/New_York";
  i18n.defaultLocale = "en_US.UTF-8";
  i18n.extraLocaleSettings = {
    LC_ADDRESS = "en_US.UTF-8";
    LC_IDENTIFICATION = "en_US.UTF-8";
    LC_MEASUREMENT = "en_US.UTF-8";
    LC_MONETARY = "en_US.UTF-8";
    LC_NAME = "en_US.UTF-8";
    LC_NUMERIC = "en_US.UTF-8";
    LC_PAPER = "en_US.UTF-8";
    LC_TELEPHONE = "en_US.UTF-8";
    LC_TIME = "en_US.UTF-8";
  };

  services.xserver.xkb = {
    layout = "us";
    variant = "";
  };

  users.users.testadmin = {
    isNormalUser = true;
    description = "Test Admin";
    extraGroups = [ "networkmanager" "wheel" ];
    packages = with pkgs; [];
  };
  services.getty.autologinUser = "testadmin";

  environment.systemPackages = with pkgs; [
    go
    tree
    tmux
  ];

  services.caddy.enable = {{.EnableCaddy}};
  services.caddy = {
    virtualHosts.":81".extraConfig = ''
      respond "Test Page"
    '';
    virtualHosts.":80".extraConfig = ''
      reverse_proxy http://localhost:8080
    '';
  };

  services.openssh.enable = {{.EnableSSH}};
  services.openssh.settings.PasswordAuthentication = true;

  networking.firewall.allowedTCPPorts = [ 80 81 8080 ];

  system.stateVersion = "24.11"; # Did you read the comment?
}`

// HTML template for the web page
const homePageTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Config Page</title>
    <script>
        function rebuildAlert() {
            alert("nixos-rebuild switch would have been applied");
        }
        function saveAlert() {
            alert("configuration.nix is being overwritten. Previous config can be found as configuration.old");
        }
    </script>
</head>
<body>
    <h2>Service Configuration</h2>
    <form action="/save" method="POST" onsubmit="saveAlert()">
        <label for="caddy">Caddy:</label>
        <select name="caddy">
            <option value="enabled" {{if .EnableCaddy}}selected{{end}}>Enabled</option>
            <option value="disabled" {{if not .EnableCaddy}}selected{{end}}>Disabled</option>
        </select>
        <br>
        <label for="ssh">SSH:</label>
        <select name="ssh">
            <option value="enabled" {{if .EnableSSH}}selected{{end}}>Enabled</option>
            <option value="disabled" {{if not .EnableSSH}}selected{{end}}>Disabled</option>
        </select>
        <br><br>
        <button type="submit">Save Config</button>
    </form>
    <form action="/apply" method="POST" onsubmit="rebuildAlert()">
        <button type="submit">Apply Config</button>
    </form>
</body>
</html>
`

type Config struct {
	EnableSSH   bool
	EnableCaddy bool
}

// Helper function to parse boolean values from the configuration file - thanks ChatGPT
func parseBooleanSetting(fileContent []byte, setting string) (bool, error) {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*%s\s*=\s*(true|false)\s*;`, setting))
	match := re.FindSubmatch(fileContent)
	if match == nil {
		return false, fmt.Errorf("%s not found", setting)
	}
	return string(match[1]) == "true", nil
}

// Save config in memory into a temp file
func (config *Config) saveTmpFile() {
	tmpl, err := texttemplate.New("nixos").Parse(nixosConfigTemplate)
	if err != nil {
		panic(err)
	}

	outFile, err := os.Create(workingDir + "configuration.tmp")
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	err = tmpl.Execute(outFile, config)
	if err != nil {
		panic(err)
	}
}

// CopyFile copies the contents of one file to another.
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

// UpdateNixosConfiguration backs up configuration.nix and replaces it with configuration.tmp.
func UpdateNixosConfiguration() error {
	configPath := workingDir + "configuration.nix"
	backupPath := workingDir + "configuration.old"
	tmpPath := workingDir + "configuration.tmp"

	// Backup existing configuration.nix to configuration.old
	fmt.Println("Backing up configuration.nix to configuration.old...")
	if err := CopyFile(configPath, backupPath); err != nil {
		return err
	}

	// Overwrite configuration.nix with configuration.tmp
	fmt.Println("Replacing configuration.nix with configuration.tmp...")
	if err := CopyFile(tmpPath, configPath); err != nil {
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

func main() {
	file, err := os.ReadFile(workingDir + "configuration.nix")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}

	// Initialize the Config struct
	config := Config{}

	// Parse the enableSSH field
	config.EnableSSH, err = parseBooleanSetting(file, "services.openssh.enable")
	if err != nil {
		fmt.Println(err)
	}

	// Parse the enableCaddy field
	config.EnableCaddy, err = parseBooleanSetting(file, "services.caddy.enable")
	if err != nil {
		fmt.Println(err)
	}

	// Output the results
	fmt.Printf("services.openssh.enable is set to %t\n", config.EnableSSH)
	fmt.Printf("services.caddy.enable is set to %t\n", config.EnableCaddy)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, _ := htmltemplate.New("webpage").Parse(homePageTemplate)
		tmpl.Execute(w, config)
	})

	http.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Failed to parse form"))
			return
		}
		config.EnableCaddy = r.FormValue("caddy") == "enabled"
		config.EnableSSH = r.FormValue("ssh") == "enabled"
		fmt.Println("Save Config Triggered", config)

		config.saveTmpFile()

		if err := UpdateNixosConfiguration(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		// w.Write([]byte("Configuration Saved"))
	})

	http.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Apply Config Triggered", config)

		// ** Uncomment below line to actually switch the nixOS config **
		// if err := applyChanges(); err != nil {
		// 	fmt.Println("Error:", err)
		// 	os.Exit(1)
		// }

		http.Redirect(w, r, "/", http.StatusSeeOther)
		// w.Write([]byte("Apply Config Triggered"))
	})

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
