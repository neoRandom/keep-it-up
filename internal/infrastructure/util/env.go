package util

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// LoadEnv reads a dotenv-style file and exports its KEY=VALUE pairs into the
// process environment. A missing file is not an error (the caller may rely on
// externally-injected environment variables), but any other I/O or scan failure
// is logged so it is not silently discarded.
func LoadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("failed to open env file %s: %v", filepath, err)
		}
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close env file %s: %v", filepath, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on the first '='
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			// Remove quotes if wrapped in them
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if err := os.Setenv(key, value); err != nil {
				log.Printf("failed to set env var %s: %v", key, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("failed to scan env file %s: %v", filepath, err)
	}
}
