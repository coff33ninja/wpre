package pipeline

type DataBus struct {
	DryRun          bool
	VaultRoot       string
	SafeFolders     []string
	ExcludePatterns []string

	ScanResults *ProfileScanResult

	TempUsername string
	TempPassword string

	TargetUsername string
	TargetSID      string

	OneDrive *OneDriveInfo
}

type ProfileScanResult struct {
	Timestamp    string       `json:"timestamp"`
	ComputerName string       `json:"computerName"`
	CurrentUser  string       `json:"currentUser"`
	Profiles     []PsProfile  `json:"profiles"`
	OneDrive     PsOneDrive   `json:"oneDrive"`
	SystemInfo   PsSystemInfo `json:"systemInfo"`
}

type PsProfile struct {
	SID    string `json:"sid"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type PsOneDrive struct {
	ProcessRunning bool     `json:"processRunning"`
	Installed      bool     `json:"installed"`
	FolderPaths    []string `json:"folderPaths"`
}

type PsSystemInfo struct {
	OSVersion  string `json:"osVersion"`
	IsElevated bool   `json:"isElevated"`
}

type OneDriveInfo struct {
	ProcessRunning bool     `json:"processRunning"`
	Accounts       []any   `json:"accounts"`
	Folders        []struct {
		Path     string  `json:"path"`
		ItemCount int    `json:"itemCount"`
		SizeMB   float64 `json:"sizeMB"`
	} `json:"folders"`
	SyncStatus string `json:"syncStatus"`
}

type TempProfileResult struct {
	Success  bool   `json:"success"`
	Username string `json:"username"`
	Action   string `json:"action"`
	Error    string `json:"error,omitempty"`
}

type OneDriveActionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type OneDrivePlaceholderResult struct {
	Path             string   `json:"path"`
	PlaceholderCount int      `json:"placeholderCount"`
	PlaceholderFiles []string `json:"placeholderFiles"`
}

func NewDataBus() *DataBus {
	return &DataBus{}
}
