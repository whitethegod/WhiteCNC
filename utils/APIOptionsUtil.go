package utils

import (
	"encoding/json"
	"fmt"
	"os"
)


func ValidateAPIOptions(options *APIOptions) error {
	
	if options.Len.Min < 1 || options.Len.Max < options.Len.Min || options.Len.Default < options.Len.Min || options.Len.Default > options.Len.Max {
		return fmt.Errorf("invalid len configuration")
	}

	
	if options.Threads.Min < 1 || options.Threads.Max < options.Threads.Min || options.Threads.Default < options.Threads.Min || options.Threads.Default > options.Threads.Max {
		return fmt.Errorf("invalid threads configuration")
	}

	
	if options.RPS.Min < 1 || options.RPS.Max < options.RPS.Min || options.RPS.Default < options.RPS.Min || options.RPS.Default > options.RPS.Max {
		return fmt.Errorf("invalid RPS configuration")
	}

	
	if options.APILoadingDelay.Enabled && (options.APILoadingDelay.DelaySeconds < 1 || options.APILoadingDelay.DelaySeconds > 10) {
		return fmt.Errorf("loading delay must be between 1 and 10 seconds")
	}

	
	if options.Geo.Default != "ALL" && !contains(options.Geo.List, options.Geo.Default) {
		return fmt.Errorf("default geo '%s' not in geo list", options.Geo.Default)
	}

	return nil
}


func GetDefaultAPIOptions() APIOptions {
	return APIOptions{
		APIGlobalCooldown: APIGlobalCooldown{
			Enabled:         true,
			PerAttackDelayMs: 100,
		},
		ShowHostInfoInAPIResponse: true,
		APILoadingDelay: APILoadingDelay{
			Enabled:       true,
			DelaySeconds:  3,
			ShowCountdown: true,
		},
		Len: APILenConfig{
			Default: 1,
			Max:     1024,
			Min:     1,
		},
		Threads: APIThreadsConfig{
			Default: 5,
			Max:     100,
			Min:     1,
		},
		RPS: APIRPSConfig{
			Default: 64,
			Max:     300,
			Min:     1,
		},
		Geo: APIGeoConfig{
			Default: "ALL",
			List:    []string{"CN", "JP", "US"},
		},
	}
}


func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}


func LoadAPIOptionsFromFile(filePath string) (APIOptions, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return GetDefaultAPIOptions(), err
	}

	var config struct {
		APIOptions APIOptions `json:"api_options"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return GetDefaultAPIOptions(), err
	}

	return config.APIOptions, nil
}