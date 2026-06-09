package orchestrator

import (
	"crypto/rand"
	"math/big"
	"os"
	"strings"
)

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func generatePassword(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			result[i] = 'X'
			continue
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

func isSystemSID(sid string) bool {
	suffixes := []string{"-500", "-501", "-502", "-18", "-19", "-20"}
	for _, s := range suffixes {
		if strings.HasSuffix(sid, s) {
			return true
		}
	}
	return false
}
