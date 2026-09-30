package cmds

import (
	"arismcnc/database"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
)

type CfxCommand struct{}

func (c *CfxCommand) Name() string {
	return "cfx"
}

func (c *CfxCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 1 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: cfx [cfx_code]\033[0m")
		return
	}

	cfxCode := strings.TrimSpace(args[0])

	
	if len(cfxCode) > 7 {
		fmt.Fprintln(output, "\033[91mLength of CFX Code is too long, please try again.\033[0m")
		return
	}

	if strings.Contains(strings.ToUpper(cfxCode), "HTTP") || strings.Contains(cfxCode, "/") || strings.Contains(cfxCode, ".") {
		fmt.Fprintln(output, "\033[91mYou need to specify an actual CFX code.\033[0m")
		return
	}

	response, err := lookupCFX(cfxCode)
	if err != nil {
		fmt.Fprintf(output, "\033[91mError fetching server data: %v\033[0m\n", err)
		return
	}

	fmt.Fprintln(output, response)
}

func (c *CfxCommand) Aliases() []string {
	return []string{"cfx"}
}

func (c *CfxCommand) AdminOnly() bool {
	return false
}


func lookupCFX(cfxCode string) (string, error) {
	apiURL := fmt.Sprintf("https://servers-frontend.fivem.net/api/servers/single/%s", cfxCode)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv102.0) Gecko/20100101 Firefox/102.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("CFX Code is not valid on FiveM database")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Hostname          string   `json:"hostname"`
			OwnerName         string   `json:"ownerName"`
			ConnectEndPoints  []string `json:"connectEndPoints"`
			Clients           int      `json:"clients"`
		} `json:"Data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("API error: %s", result.Error)
	}

	if len(result.Data.ConnectEndPoints) == 0 {
		return "", fmt.Errorf("no connection endpoints found")
	}

	
	endpoint := result.Data.ConnectEndPoints[0]
	parts := strings.Split(endpoint, ":")
	serverIP := parts[0]
	serverPort := ""
	if len(parts) > 1 {
		serverPort = parts[1]
	}

	serverJSON := fmt.Sprintf("http://%s/players.json", endpoint)

	response := fmt.Sprintf(
		"\033[91mCFX Lookup result:\n"+
			"Server Name: %s\n"+
			"Server Owner: %s\n"+
			"Server IP: %s\n"+
			"Server Port: %s\n"+
			"Server JSON: %s\n"+
			"Connected Players: %d\033[0m",
		result.Data.Hostname,
		result.Data.OwnerName,
		serverIP,
		serverPort,
		serverJSON,
		result.Data.Clients,
	)

	return response, nil
}

func init() {
	CommandMap["cfx"] = &CfxCommand{}
}