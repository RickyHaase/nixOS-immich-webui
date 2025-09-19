package main

import (
	"log/slog"
	"net/http"

	"github.com/RickyHaase/nixOS-immich-webui/internal/handlers"
	"github.com/RickyHaase/nixOS-immich-webui/internal/services"
	"github.com/RickyHaase/nixOS-immich-webui/internal/templates"
)

func main() {
	// Initialize services
	backupService := services.NewBackupService()

	// Initialize handlers
	systemHandler := handlers.NewSystemHandler(templates.FS)
	immichHandler := handlers.NewImmichHandler(templates.FS)
	backupHandler := handlers.NewBackupHandler(templates.FS, backupService)

	// Setup HTTP routes
	mux := http.NewServeMux()
	
	// System routes
	mux.HandleFunc("GET /{$}", systemHandler.HandleRoot)
	mux.HandleFunc("POST /save", systemHandler.HandleSave)
	mux.HandleFunc("POST /apply", systemHandler.HandleApply)
	mux.HandleFunc("POST /poweroff", systemHandler.HandlePoweroff)
	mux.HandleFunc("POST /reboot", systemHandler.HandleReboot)
	
	// Immich routes
	mux.HandleFunc("GET /status", immichHandler.HandleStatus)
	mux.HandleFunc("POST /stop", immichHandler.HandleStop)
	mux.HandleFunc("POST /start", immichHandler.HandleStart)
	mux.HandleFunc("POST /update", immichHandler.HandleUpdate)
	mux.HandleFunc("POST /email", immichHandler.HandleEmailPost)
	
	// Backup routes
	mux.HandleFunc("GET /disks", backupHandler.HandleGetDisks)
	mux.HandleFunc("POST /backup", backupHandler.HandleBackup)
	mux.HandleFunc("GET /backupstatus", backupHandler.HandleGetBackupStatus)

	// Debug mode configuration (uncomment to enable)
	// slog.SetLogLoggerLevel(slog.LevelDebug)

	slog.Info("Server started at http://localhost:8000")
	if err := http.ListenAndServe("localhost:8000", mux); err != nil {
		slog.Error("Server failed", "err", err)
	}
}