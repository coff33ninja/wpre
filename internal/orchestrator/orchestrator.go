package orchestrator

import (
	"flag"
	"fmt"
	"os"

	"wpre/internal/config"
	"wpre/internal/executor"
	"wpre/internal/logging"
	"wpre/internal/pipeline"
	"wpre/internal/reboot"
	"wpre/resources"
	"wpre/internal/rollback"
	"wpre/internal/state"
)

func Run() int {
	cfgPath := flag.String("config", "wpre.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Scan and simulate without making changes")
	safeMode := flag.Bool("safe-mode", false, "Skip dangerous operations")
	resumeStage := flag.String("resume", "", "Resume from a specific stage")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		cfg = config.Default()
	}
	if *dryRun {
		cfg.Safety.DryRun = true
	}
	if *safeMode {
		cfg.Safety.SafeMode = true
	}

	logDir := "logs"
	logger, err := logging.New(logging.LevelInfo, logDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		return 99
	}
	defer logger.Close()

	logger.Info("WPRE starting — version %s", cfg.Version)
	logger.Info("config: %s", *cfgPath)

	moduleRoot, err := resources.ModuleBasePath()
	if err != nil {
		logger.Error("failed to extract embedded modules: %v", err)
		return 99
	}
	logger.Info("modules extracted to: %s", moduleRoot)

	if cfg.Safety.DryRun {
		logger.Warn("DRY RUN MODE — no changes will be made")
	}

	rollbackMgr, err := rollback.New("rollback")
	if err != nil {
		logger.Error("failed to initialize rollback manager: %v", err)
		return 99
	}

	rebootMgr := reboot.NewManager()
	_ = rebootMgr

	ps := executor.NewPowerShellRunner(moduleRoot)
	py := executor.NewPythonRunner(moduleRoot)

	data := pipeline.NewDataBus()
	psState := state.New()
	pipe := pipeline.New()

	ctx := &pipeline.Context{
		State:    psState,
		Config:   cfg,
		Logger:   logger,
		Rollback: rollbackMgr,
		Data:     data,
		RunPS: func(args ...string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("RunPS requires at least script path")
			}
			result, err := ps.RunScript(args[0], args[1:]...)
			if err != nil {
				return "", fmt.Errorf("PS script error (exit=%d): %s\nstderr: %s",
					result.ExitCode, args[0], result.Stderr)
			}
			return result.Stdout, nil
		},
		RunPY: func(args ...string) (string, error) {
			if len(args) < 1 {
				return "", fmt.Errorf("RunPY requires at least script path")
			}
			result, err := py.RunScript(args[0], args[1:]...)
			if err != nil {
				return "", fmt.Errorf("PY script error (exit=%d): %s\nstderr: %s",
					result.ExitCode, args[0], result.Stderr)
			}
			return result.Stdout, nil
		},
	}

	registerHandlers(pipe)

	if *resumeStage != "" {
		startStage := stageFromString(*resumeStage)
		logger.Info("resuming from stage: %s", *resumeStage)
		if err := pipe.RunFrom(ctx, startStage); err != nil {
			logger.Error("pipeline failed: %v", err)
			return 2
		}
	} else {
		if err := pipe.Run(ctx); err != nil {
			logger.Error("pipeline failed: %v", err)
			return 2
		}
	}

	logger.Info("WPRE completed successfully")
	return 0
}

func registerHandlers(p *pipeline.Pipeline) {
	p.Register(state.StageBootstrap, handleBootstrap)
	p.Register(state.StageProfileScan, handleProfileScan)
	p.Register(state.StageBackupExisting, handleBackupExisting)
	p.Register(state.StageTempProfileCreate, handleTempProfileCreate)
	p.Register(state.StageDataHarvest, handleDataHarvest)
	p.Register(state.StageOneDriveExtract, handleOneDriveExtract)
	p.Register(state.StageAppCapture, handleAppCapture)
	p.Register(state.StageNewProfileCreate, handleNewProfileCreate)
	p.Register(state.StageDataInject, handleDataInject)
	p.Register(state.StageFirstLoginInit, handleFirstLoginInit)
	p.Register(state.StageProfileCleanup, handleProfileCleanup)
	p.Register(state.StageFinalValidation, handleFinalValidation)
	p.Register(state.StageComplete, handleComplete)
}

func stageFromString(s string) state.Stage {
	stages := map[string]state.Stage{
		"bootstrap":           state.StageBootstrap,
		"profile_scan":        state.StageProfileScan,
		"backup_existing":     state.StageBackupExisting,
		"temp_profile_create": state.StageTempProfileCreate,
		"data_harvest":        state.StageDataHarvest,
		"onedrive_extract":    state.StageOneDriveExtract,
		"app_capture":         state.StageAppCapture,
		"new_profile_create":  state.StageNewProfileCreate,
		"data_inject":         state.StageDataInject,
		"first_login_init":    state.StageFirstLoginInit,
		"profile_cleanup":     state.StageProfileCleanup,
		"final_validation":    state.StageFinalValidation,
		"complete":            state.StageComplete,
	}
	if s, ok := stages[s]; ok {
		return s
	}
	return state.StageBootstrap
}
