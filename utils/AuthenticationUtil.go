package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)


func Authenticate() bool {

	
	fail := func(msg string) bool {
		log.Printf("\033[31m%s\033[0m\n", msg)
		os.Exit(1)
		return false
	}

	
	config, err := LoadConfig("assets/config.json")
	if err != nil {
		return fail("Failed to load config.json")
	}

	license := config.License
	if license == "" {
		return fail("No license found in config.json")
	}

	
	resp, err := http.Get("http://185.14.92.194/cnc/1.php?action=authentication&license=" + license)
	if err != nil {
		return fail("Unable to reach authentication server")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail("Authentication server returned an invalid response")
	}

	
	type LicenseResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			LicenseKey string `json:"license_key"`
			ExpiryDate string `json:"expirydate"`
			Banned     int    `json:"banned"`
		} `json:"data"`
	}

	var lic LicenseResponse
	err = json.Unmarshal(body, &lic)
	if err != nil {
		return fail("Authentication server sent invalid JSON format")
	}

	
	switch lic.Status {

	case "invalid":
		return fail("Invalid license key")

	case "banned":
		return fail("This license is BANNED")

	case "expired":
		return fail("This license has EXPIRED")

	case "error":
		return fail("Authentication server returned an error")

	case "valid":
		
	default:
		return fail("Unexpected authentication response")
	}

	
	expiry, err := time.Parse("2006-01-02", lic.Data.ExpiryDate)
	if err != nil {
		return fail("Invalid expiry date from server")
	}

	now := time.Now()
	diff := expiry.Sub(now)

	if diff <= 0 {
		return fail("This license has EXPIRED")
	}

	days := int(diff.Hours()) / 24
	hours := int(diff.Hours()) % 24

	
	if days <= 7 {
		log.Printf("\033[33mLicense will expire in %d days, %d hours.\033[0m\n", days, hours)
	} else {
		log.Printf("\033[32mLicense validated. Expires in %d days, %d hours.\033[0m\n", days, hours)
	}

	return true
}


func GetPublicIP() (string, error) {
	resp, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		return "", fmt.Errorf("failed to get public IP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK response retrieving IP: %v", resp.StatusCode)
	}

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed reading public IP: %v", err)
	}

	return string(ip), nil
}
