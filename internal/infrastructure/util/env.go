package util

import (
	"bufio"
	"os"
	"strings"
)

func LoadEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return // Silently fail if no .env file exists (e.g., in production)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Err() != nil {
			return
		}

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
			os.Setenv(key, value)
		}
	}
}
