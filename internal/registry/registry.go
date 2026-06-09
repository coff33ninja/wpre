package registry

import (
	"fmt"
	"strings"
	"syscall"
	"golang.org/x/sys/windows/registry"
)

type Key struct {
	Path string
	Key  registry.Key
}

func OpenKey(path string) (*Key, error) {
	parts := strings.SplitN(path, "\\", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid registry path: %s", path)
	}

	rootKey, err := parseRootKey(parts[0])
	if err != nil {
		return nil, err
	}

	k, err := registry.OpenKey(rootKey, parts[1], registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry key %s: %w", path, err)
	}

	return &Key{Path: path, Key: k}, nil
}

func parseRootKey(name string) (registry.Key, error) {
	switch strings.ToUpper(name) {
	case "HKEY_LOCAL_MACHINE", "HKLM":
		return registry.LOCAL_MACHINE, nil
	case "HKEY_CURRENT_USER", "HKCU":
		return registry.CURRENT_USER, nil
	case "HKEY_USERS", "HKU":
		return registry.USERS, nil
	case "HKEY_CLASSES_ROOT", "HKCR":
		return registry.CLASSES_ROOT, nil
	case "HKEY_CURRENT_CONFIG", "HKCC":
		return registry.CURRENT_CONFIG, nil
	default:
		return 0, fmt.Errorf("unknown root key: %s", name)
	}
}

func (k *Key) GetString(name string) (string, error) {
	val, _, err := k.Key.GetStringValue(name)
	if err != nil {
		return "", fmt.Errorf("failed to read %s\\%s: %w", k.Path, name, err)
	}
	return val, nil
}

func (k *Key) GetInteger(name string) (uint64, error) {
	val, _, err := k.Key.GetIntegerValue(name)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s\\%s: %w", k.Path, name, err)
	}
	return val, nil
}

func (k *Key) GetStrings(name string) ([]string, error) {
	val, _, err := k.Key.GetStringsValue(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s\\%s: %w", k.Path, name, err)
	}
	return val, nil
}

func (k *Key) SubKeyNames() ([]string, error) {
	names, err := k.Key.ReadSubKeyNames(0)
	if err != nil {
		return nil, fmt.Errorf("failed to read subkeys of %s: %w", k.Path, err)
	}
	return names, nil
}

func (k *Key) ValueNames() ([]string, error) {
	names, err := k.Key.ReadValueNames(0)
	if err != nil {
		return nil, fmt.Errorf("failed to read value names of %s: %w", k.Path, err)
	}
	return names, nil
}

func (k *Key) Close() error {
	return k.Key.Close()
}

func IsOneDriveInstalled() bool {
	k, err := OpenKey("HKCU\\Software\\Microsoft\\OneDrive\\Accounts")
	if err != nil {
		return false
	}
	defer k.Close()
	return true
}

func GetOneDriveAccounts() ([]string, error) {
	k, err := OpenKey("HKCU\\Software\\Microsoft\\OneDrive\\Accounts")
	if err != nil {
		return nil, err
	}
	defer k.Close()
	return k.SubKeyNames()
}

func GetCurrentUserSID() (string, error) {
	k, err := OpenKey("HKCU\\Volatile Environment")
	if err != nil {
		return "", err
	}
	defer k.Close()
	return k.GetString("USERNAME")
}

func SIDFromProfilePath(profilePath string) (string, error) {
	// HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList
	k, err := OpenKey("HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\ProfileList")
	if err != nil {
		return "", err
	}
	defer k.Close()
	_ = profilePath

	sids, err := k.SubKeyNames()
	if err != nil {
		return "", err
	}
	for _, sid := range sids {
		sidKey, err := OpenKey(fmt.Sprintf("HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\ProfileList\\%s", sid))
		if err != nil {
			continue
		}
		profilePathVal, err := sidKey.GetString("ProfileImagePath")
		sidKey.Close()
		if err != nil {
			continue
		}
		if strings.EqualFold(profilePathVal, profilePath) {
			return sid, nil
		}
	}
	return "", fmt.Errorf("SID not found for profile path: %s", profilePath)
}

func ExportKey(path, outputFile string) error {
	// Uses reg.exe export for simplicity
	cmd := syscall.StringToUTF16Ptr(fmt.Sprintf("reg.exe export \"%s\" \"%s\" /y", path, outputFile))
	si := new(syscall.StartupInfo)
	pi := new(syscall.ProcessInformation)

	err := syscall.CreateProcess(
		nil, cmd, nil, nil, false, 0, nil, nil, si, pi,
	)
	if err != nil {
		return fmt.Errorf("failed to export registry key: %w", err)
	}
	syscall.WaitForSingleObject(pi.Process, syscall.INFINITE)
	syscall.CloseHandle(pi.Process)
	syscall.CloseHandle(pi.Thread)

	return nil
}
