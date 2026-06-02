package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/metrics"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/orchestrator"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
)

func main() {
	log := logger.New()

	if err := run(log); err != nil {
		log.Error("Application error", logger.Error(err))
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	ctx := context.Background()

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		if sa := os.Getenv("SERVICE_ACCOUNT_PATH"); sa != "" {
			_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", sa)
		}
	}

	var appMetrics *metrics.Metrics
	meter, shutdownMetrics, err := metrics.InitProvider(ctx)
	if err != nil {
		log.Warn("Metrics initialisation failed, continuing without metrics", logger.Error(err))
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if serr := shutdownMetrics(shutdownCtx); serr != nil {
				log.Warn("Failed to flush metrics on shutdown", logger.Error(serr))
			}
		}()
		appMetrics, err = metrics.New(meter)
		if err != nil {
			log.Warn("Failed to create metric instruments, continuing without metrics", logger.Error(err))
			appMetrics = nil
		} else {
			log.Info("Metrics initialized", logger.Status("ready"))
		}
	}

	configPath := getEnvOrDefault("CONFIG_PATH", "./data/user_config.toml")
	featureCfg, err := service.LoadFeatureConfig(configPath)
	if err != nil {
		log.Error("Failed to load feature config", logger.Error(err), logger.F("path", configPath))
		return err
	}

	infraCfg, err := config.LoadWithFile(resolveEnvFile())
	if err != nil {
		log.Error("Failed to load .env", logger.Error(err))
		return err
	}

	calendarSvc, err := service.NewCalendarService(ctx, featureCfg.Calendar)
	if err != nil {
		log.Error("Failed to initialize calendar service", logger.Error(err))
		return err
	}

	vmwareSvc, err := service.NewVMwareService(ctx, infraCfg, log)
	if err != nil {
		log.Error("Failed to initialize VMware service", logger.Error(err))
		return err
	}

	var emailSvc service.EmailSender
	smtpHost := getEnvOrDefault("SMTP_HOST", "smtp.gmail.com")
	smtpPort := getEnvOrDefault("SMTP_PORT", "587")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if smtpPasswordFile := os.Getenv("SMTP_PASSWORD_FILE"); smtpPasswordFile != "" {
		passwordBytes, err := os.ReadFile(smtpPasswordFile)
		if err != nil {
			log.Warn("Failed to read SMTP password file", logger.Error(err), logger.F("file", smtpPasswordFile))
		} else {
			smtpPassword = strings.TrimSpace(string(passwordBytes))
		}
	}
	smtpFrom := getEnvOrDefault("SMTP_FROM", smtpUsername)
	testEmailOnly := os.Getenv("TEST_EMAIL_ONLY")
	if smtpUsername != "" && smtpPassword != "" {
		svc, err := service.NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, smtpFrom, testEmailOnly)
		if err != nil {
			log.Warn("Email service not available, emails will not be sent", logger.Error(err))
		} else {
			emailSvc = svc
			if testEmailOnly != "" {
				log.Info("Email service initialized (TEST MODE)", logger.Status("ready"), logger.F("TEST_EMAIL", testEmailOnly))
			} else {
				log.Info("Email service initialized", logger.Status("ready"))
			}
		}
	} else {
		log.Info("Email service not configured (SMTP_USERNAME/SMTP_PASSWORD missing)")
	}

	var wireguardSvc service.WireGuardManager
	if featureCfg.WireGuard.Enabled {
		featureCfg.WireGuard.OPNsenseAPIKey = os.Getenv("OPNSENSE_API_KEY")
		featureCfg.WireGuard.OPNsenseAPISecret = os.Getenv("OPNSENSE_API_SECRET")

		var opnsense service.OPNsenseAPI
		if featureCfg.WireGuard.AutoRegisterPeers && featureCfg.WireGuard.OPNsenseURL != "" && featureCfg.WireGuard.OPNsenseAPIKey != "" {
			opnsense = service.NewOPNsenseClient(
				featureCfg.WireGuard.OPNsenseURL,
				featureCfg.WireGuard.OPNsenseAPIKey,
				featureCfg.WireGuard.OPNsenseAPISecret,
				nil,
				featureCfg.WireGuard.OPNsenseInsecure,
			)
		}

		wgSvc := service.NewWireGuardService(&featureCfg.WireGuard, opnsense)
		if err := wgSvc.ValidateConfig(); err != nil {
			log.Warn("WireGuard service configuration invalid", logger.Error(err))
		} else {
			wireguardSvc = wgSvc
			log.Info("WireGuard service initialized", logger.Status("ready"))
		}
	} else {
		log.Info("WireGuard service not enabled in configuration")
	}

	orch := &orchestrator.Orchestrator{
		Logger:     log,
		Calendar:   calendarSvc,
		VMware:     vmwareSvc,
		Email:      emailSvc,
		WireGuard:  wireguardSvc,
		FeatureCfg: featureCfg,
		Metrics:    appMetrics,
	}

	return orch.Run()
}

func resolveEnvFile() string {
	if path := os.Getenv("ENV_PATH"); path != "" {
		return path
	}
	for _, path := range []string{".env", "../.env"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ".env"
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
