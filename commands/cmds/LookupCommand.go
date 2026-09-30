package cmds

import (
	"arismcnc/database"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gliderlabs/ssh"
)

type LookupCommand struct{}

func (l *LookupCommand) Name() string {
	return "lookup"
}

func (l *LookupCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 1 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: lookup [ip/url]\033[0m")
		return
	}

	target := strings.TrimSpace(args[0])

	if !isValidIP(target) && !isValidURL(target) {
		fmt.Fprintln(output, "\033[91mInvalid target. Provide a valid IP or URL.\033[0m")
		return
	}

	response, err := lookup(target)
	if err != nil {
		fmt.Fprintf(output, "\033[91mError performing lookup: %v\033[0m\n", err)
		return
	}

	fmt.Fprintln(output, response)
}

func (l *LookupCommand) Aliases() []string {
	return []string{"lookup"}
}

func (l *LookupCommand) AdminOnly() bool {
	return false
}


func lookup(target string) (string, error) {
	
	resolvedTarget := target
	if isValidURL(target) && !isValidIP(target) {
		ip, err := resolveURLToIP(target)
		if err != nil {
			return "", fmt.Errorf("failed to resolve URL: %w", err)
		}
		resolvedTarget = ip
	}

	
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,isp,org,as,query,zip,timezone,lat,lon", url.QueryEscape(resolvedTarget))

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var result struct {
		Status      string  `json:"status"`
		Message     string  `json:"message"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		AS          string  `json:"as"`
		Zip         string  `json:"zip"`
		Timezone    string  `json:"timezone"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Query       string  `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	
	if result.Status != "success" {
		return "", fmt.Errorf("ip-api.com returned error: %s", result.Message)
	}

	
	organization := result.Org
	if organization == "" {
		organization = result.ISP
	}
	if organization == "" {
		organization = "Unknown"
	}

	
	response := fmt.Sprintf(
		"\033[91mLookup result:\n"+
			"Host: %s\n"+
			"IP: %s\n"+
			"Country: %s (%s)\n"+
			"Region: %s\n"+
			"City: %s\n"+
			"ISP: %s\n"+
			"Organization: %s\n"+
			"ASN: %s\n"+
			"Timezone: %s\n"+
			"Coordinates: %.6f, %.6f\n"+
			"ZIP: %s\033[0m",
		target,
		result.Query,
		result.Country,
		result.CountryCode,
		result.RegionName,
		result.City,
		result.ISP,
		organization,
		result.AS,
		result.Timezone,
		result.Lat,
		result.Lon,
		result.Zip,
	)

	return response, nil
}


func resolveURLToIP(domain string) (string, error) {
	
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}

	parsed, err := url.Parse(domain)
	if err != nil {
		return "", err
	}

	
	host := parsed.Hostname()
	if host == "" {
		host = parsed.Path
	}

	
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}

	if len(addrs) == 0 {
		return "", fmt.Errorf("no IP addresses found for %s", domain)
	}

	
	return addrs[0], nil
}

func isValidIP(ip string) bool {
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	return ipRegex.MatchString(ip)
}

func isValidURL(input string) bool {
	if parsed, err := url.ParseRequestURI(input); err == nil {
		if strings.HasPrefix(parsed.Scheme, "http") {
			return true
		}
	}

	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?(\.[a-zA-Z]{2,})+$`)
	return domainRegex.MatchString(input)
}

func init() {
	CommandMap["lookup"] = &LookupCommand{}
}