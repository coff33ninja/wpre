package state

type Stage int

const (
	StageBootstrap Stage = iota
	StageProfileScan
	StageBackupExisting
	StageTempProfileCreate
	StageDataHarvest
	StageOneDriveExtract
	StageAppCapture
	StageNewProfileCreate
	StageDataInject
	StageFirstLoginInit
	StageProfileCleanup
	StageFinalValidation
	StageComplete
)

func (s Stage) String() string {
	switch s {
	case StageBootstrap:
		return "bootstrap"
	case StageProfileScan:
		return "profile_scan"
	case StageBackupExisting:
		return "backup_existing"
	case StageTempProfileCreate:
		return "temp_profile_create"
	case StageDataHarvest:
		return "data_harvest"
	case StageOneDriveExtract:
		return "onedrive_extract"
	case StageAppCapture:
		return "app_capture"
	case StageNewProfileCreate:
		return "new_profile_create"
	case StageDataInject:
		return "data_inject"
	case StageFirstLoginInit:
		return "first_login_init"
	case StageProfileCleanup:
		return "profile_cleanup"
	case StageFinalValidation:
		return "final_validation"
	case StageComplete:
		return "complete"
	default:
		return "unknown"
	}
}

type PipelineState struct {
	CurrentStage Stage
	Completed    []Stage
	Failed       []StageError
	ManifestPath string
}

type StageError struct {
	Stage   Stage  `json:"stage"`
	Message string `json:"message"`
	Fatal   bool   `json:"fatal"`
}

func New() *PipelineState {
	return &PipelineState{
		CurrentStage: StageBootstrap,
		Completed:    make([]Stage, 0),
		Failed:       make([]StageError, 0),
	}
}

func (ps *PipelineState) Advance(next Stage) {
	ps.Completed = append(ps.Completed, ps.CurrentStage)
	ps.CurrentStage = next
}

func (ps *PipelineState) AddError(err StageError) {
	ps.Failed = append(ps.Failed, err)
}

func (ps *PipelineState) IsComplete() bool {
	return ps.CurrentStage == StageComplete
}

func (ps *PipelineState) HasFatalErrors() bool {
	for _, e := range ps.Failed {
		if e.Fatal {
			return true
		}
	}
	return false
}
