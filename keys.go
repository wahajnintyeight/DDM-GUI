package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// depotKeyFor returns the 32-byte key for one depot from a keys.txt file ("depotID;hexKey" per line).
// The key is never included in an error message.
func depotKeyFor(keysPath, depotID string) ([]byte, error) {
	file, err := os.Open(keysPath)
	if err != nil {
		return nil, fmt.Errorf("keys.txt could not be opened: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var found []byte
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\xef\xbb\xbf"))
		id, value, ok := strings.Cut(line, ";")
		if !ok || strings.TrimSpace(id) != depotID {
			continue
		}
		key, err := hex.DecodeString(strings.TrimSpace(value))
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("keys.txt has an invalid key for depot %s (expected 64 hexadecimal characters)", depotID)
		}
		found = key
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("keys.txt could not be read: %w", err)
	}
	if found == nil {
		return nil, errors.New("keys.txt has no key for depot " + depotID)
	}
	return found, nil
}
