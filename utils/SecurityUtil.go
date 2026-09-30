package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	"net/http"
	"bytes"
)


type LogManager struct {
	Config LogConfig
	mu     sync.RWMutex
}

type LogConfig struct {
	Global struct {
		Enabled      bool `json:"enabled"`
		LogInFiles   bool `json:"log_in_files"`
	} `json:"global"`
	Discord struct {
		Enabled     bool   `json:"enabled"`
		WebhookURL  string `json:"webhook_url"`
	} `json:"discord"`
	DiscordIntegration struct {
		BlacklistedHost struct {
			WebhookURL string `json:"webhook_url"`
			Title      string `json:"title"`
			Image      string `json:"image"`
			HexColor   string `json:"hex_color"`
			Enabled    bool   `json:"enabled"`
		} `json:"Blacklisted_Host"`
	} `json:"Discord_Integration"`
}

type DiscordWebhook struct {
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
	Timestamp   string `json:"timestamp"`
	Footer      struct {
		Text string `json:"text"`
	} `json:"footer"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
}

type SecuritySettings struct {
	MaxConnectionsPerIP  IPRateLimitConfig  `json:"Max_Connections_Per_IP"`
	BlockedSSHClients    []string           `json:"Blocked_SSH_Clients"`
	WhitelistedSSHClients WhitelistConfig   `json:"Whitelisted_SSH_Clients"`
	WhitelistCNCIP       IPWhitelistConfig  `json:"Whitelist_CNC_IP"`
	WhitelistAPIIP       IPWhitelistConfig  `json:"Whitelist_IP_FOR_API"`
	BlacklistIP          IPBlacklistConfig  `json:"Blacklist_IP_CNC_API"`
	mu                   sync.RWMutex       `json:"-"`
	
	
	LogConfig           LogConfig           `json:"-"`
}




func NewLogManager(configPath string) (*LogManager, error) {
	lm := &LogManager{}
	err := lm.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	return lm, nil
}


func (lm *LogManager) LoadConfig(configPath string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&lm.Config); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	return nil
}


func (lm *LogManager) SendBlacklistedHostAttackWebhook(attackerIP, targetHost, username string) {
	if !lm.Config.Global.Enabled {
		return
	}

	if !lm.Config.DiscordIntegration.BlacklistedHost.Enabled {
		return
	}

	webhookURL := lm.Config.DiscordIntegration.BlacklistedHost.WebhookURL
	if webhookURL == "" {
		log.Printf("[WEBHOOK] Webhook URL not configured for blacklisted host alerts")
		return
	}

	
	embed := DiscordEmbed{
		Title:       lm.Config.DiscordIntegration.BlacklistedHost.Title,
		Description: fmt.Sprintf("**User:** `%s`\n**Attacker IP:** `%s`\n**Target Host:** `%s`\n\nÃ°ÂÂÂ¨ **Blacklisted Host Attack Attempt** Ã°ÂÂÂ¨", username, attackerIP, targetHost),
		Color:       hexToDecimal(lm.Config.DiscordIntegration.BlacklistedHost.HexColor),
		Timestamp:   time.Now().Format(time.RFC3339),
		Image: struct {
			URL string `json:"url"`
		}{
			URL: lm.Config.DiscordIntegration.BlacklistedHost.Image,
		},
	}
	
	embed.Footer.Text = "Security Alert Ã¢ÂÂ¢ " + time.Now().Format("2006-01-02 15:04:05 MST")

	webhook := DiscordWebhook{
		Username:  "Security Bot",
		AvatarURL: "https://cdn-icons-png.flaticon.com/512/2784/2784449.png",
		Embeds:    []DiscordEmbed{embed},
	}

	
	go lm.sendDiscordWebhook(webhookURL, webhook)
}


func (lm *LogManager) sendDiscordWebhook(webhookURL string, webhook DiscordWebhook) {
	jsonData, err := json.Marshal(webhook)
	if err != nil {
		log.Printf("[WEBHOOK] Error marshaling webhook data: %v", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[WEBHOOK] Error sending webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[WEBHOOK] Blacklisted host attack alert sent successfully")
	} else {
		log.Printf("[WEBHOOK] Failed to send webhook. Status: %d", resp.StatusCode)
	}
}


func hexToDecimal(hex string) int {
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}
	
	var color int
	_, err := fmt.Sscanf(hex, "%x", &color)
	if err != nil {
		return 16711680 
	}
	return color
}




func (s *SecuritySettings) IsIPBlacklisted(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.BlacklistIP.Enabled {
		return false
	}

	
	ipOnly := strings.Split(ip, ":")[0]

	for _, blockedIP := range s.BlacklistIP.IPs {
		if ipOnly == blockedIP {
			log.Printf("[SECURITY] Ã¢ÂÂ Blacklisted IP detected: %s", ipOnly)
			
			
			go s.sendBlacklistIPDetectedWebhook(ipOnly)
			return true
		}
	}

	return false
}



func (s *SecuritySettings) ValidateHostAttack(attackerIP, targetHost, username string) (bool, string) {
	
	
	
	
	
	
	
	hostBlacklist := []string{
		"gov.br",
		"microsoft.com",
		"apple.com",
		
	}
	
	
	for _, blockedHost := range hostBlacklist {
		if strings.Contains(strings.ToLower(targetHost), strings.ToLower(blockedHost)) {
			log.Printf("[SECURITY] Ã¢ÂÂ Blacklisted host attack attempt by %s (%s) to %s", 
				username, attackerIP, targetHost)
			
			
			go s.sendBlacklistedHostAttackWebhook(attackerIP, targetHost, username)
			
			return false, fmt.Sprintf("Host '%s' is blacklisted", targetHost)
		}
	}
	
	return true, ""
}




func (s *SecuritySettings) sendBlacklistIPDetectedWebhook(ip string) {
	
	lm, err := NewLogManager("./assets/logs/logs.json")
	if err != nil {
		log.Printf("[WEBHOOK] Error creating LogManager: %v", err)
		return
	}
	
	
	lm.SendBlacklistedHostAttackWebhook(ip, "Blacklisted IP Detected", "System")
}


func (s *SecuritySettings) sendBlacklistedHostAttackWebhook(attackerIP, targetHost, username string) {
	
	lm, err := NewLogManager("./assets/logs/logs.json")
	if err != nil {
		log.Printf("[WEBHOOK] Error creating LogManager: %v", err)
		return
	}
	
	
	lm.SendBlacklistedHostAttackWebhook(attackerIP, targetHost, username)
}



type WhitelistConfig struct {
	Enabled bool     `json:"enabled"`
	List    []string `json:"list"`
}

type IPWhitelistConfig struct {
	Enabled bool     `json:"enabled"`
	IPs     []string `json:"IPs"`
}

type IPBlacklistConfig struct {
	Enabled bool     `json:"enabled"`
	IPs     []string `json:"IPs"`
}

type ConfigWithSecurity struct {
	CNC struct {
		License string `json:"license"`
		Port    string `json:"port"`
		APIPort string `json:"api_port"`
	} `json:"cnc"`

	SecuritySettings SecuritySettings `json:"Security_Settings"`
}

var (
	securityConfig     *SecuritySettings
	securityConfigOnce sync.Once
)

func LoadFullConfig(filePath string) (*ConfigWithSecurity, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg ConfigWithSecurity
	err = json.NewDecoder(f).Decode(&cfg)
	return &cfg, err
}


func (s *SecuritySettings) ValidateIPOnly(ip string) (bool, string) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    
    ipOnly := strings.Split(ip, ":")[0]

    
    if s.BlacklistIP.Enabled {
        for _, blockedIP := range s.BlacklistIP.IPs {
            if ipOnly == blockedIP {
				
				go s.sendBlacklistIPDetectedWebhook(ip)
                return false, fmt.Sprintf("IP %s is blacklisted", ip)
            }
        }
    }

    
    if s.WhitelistCNCIP.Enabled {
        allowed := false
        for _, allowedIP := range s.WhitelistCNCIP.IPs {
            if ipOnly == allowedIP {
                allowed = true
                break
            }
        }
        if !allowed {
            return false, fmt.Sprintf("IP %s is not whitelisted for CNC access", ip)
        }
    }

    return true, ""
}


func GetSecurityConfig() *SecuritySettings {
	securityConfigOnce.Do(func() {
		cfg, err := LoadFullConfig("assets/config.json")
		if err != nil {
			log.Println("[SECURITY] default config")
			securityConfig = &SecuritySettings{
				MaxConnectionsPerIP: IPRateLimitConfig{
					Enabled:           true,
					MaxConnections:    3,
					TimeWindowMinutes: 1,
					CooldownMinutes:   1,
				},
			}
		} else {
			securityConfig = &cfg.SecuritySettings
		}
	})
	return securityConfig
}


func LoadSecuritySettings(filePath string) (*SecuritySettings, error) {
    securityConfigOnce.Do(func() {
        file, err := os.Open(filePath)
        if err != nil {
            log.Printf("[SECURITY] Failed to open config: %v", err)
            securityConfig = &SecuritySettings{}
            return
        }
        defer file.Close()

        var fullConfig ConfigWithSecurity
        decoder := json.NewDecoder(file)
        if err := decoder.Decode(&fullConfig); err != nil {
            log.Printf("[SECURITY] Failed to decode config: %v", err)
            securityConfig = &SecuritySettings{}
            return
        }

        
        securityConfig = &fullConfig.SecuritySettings

        
    })

    return securityConfig, nil
}



func ReloadSecuritySettings(filePath string) (*SecuritySettings, error) {
	securityConfigOnce = sync.Once{} 
	return LoadSecuritySettings(filePath)
}


func (s *SecuritySettings) IsSSHClientBlocked(clientVersion string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	
	clientVersion = strings.TrimSpace(clientVersion)

	
	for _, blocked := range s.BlockedSSHClients {
		if strings.Contains(clientVersion, blocked) {
			log.Printf("[SECURITY] Ã¢ÂÂ Blocked SSH client detected: %s (matched: %s)", clientVersion, blocked)
			return true
		}
	}

	return false
}


func (s *SecuritySettings) IsSSHClientWhitelisted(clientVersion string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	
	if !s.WhitelistedSSHClients.Enabled {
		return true
	}

	clientVersion = strings.TrimSpace(clientVersion)

	
	for _, allowed := range s.WhitelistedSSHClients.List {
		if strings.Contains(clientVersion, allowed) {
			log.Printf("[SECURITY] Ã¢ÂÂ Whitelisted SSH client: %s", clientVersion)
			return true
		}
	}

	log.Printf("[SECURITY] Ã¢ÂÂ SSH client not in whitelist: %s", clientVersion)
	return false
}


func (s *SecuritySettings) ValidateSSHClient(clientVersion string) (bool, string) {
	
	if s.IsSSHClientBlocked(clientVersion) {
		return false, fmt.Sprintf("SSH client '%s' is blocked", clientVersion)
	}

	
	if !s.IsSSHClientWhitelisted(clientVersion) {
		return false, fmt.Sprintf("SSH client '%s' is not whitelisted", clientVersion)
	}

	return true, ""
}


func (s *SecuritySettings) IsIPWhitelistedForCNC(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.WhitelistCNCIP.Enabled {
		return true 
	}

	ip = strings.Split(ip, ":")[0]

	for _, allowedIP := range s.WhitelistCNCIP.IPs {
		if ip == allowedIP {
			log.Printf("[SECURITY] Ã¢ÂÂ Whitelisted CNC IP: %s", ip)
			return true
		}
	}

	log.Printf("[SECURITY] Ã¢ÂÂ IP not whitelisted for CNC: %s", ip)
	return false
}


func (s *SecuritySettings) IsIPWhitelistedForAPI(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.WhitelistAPIIP.Enabled {
		return true 
	}

	ip = strings.Split(ip, ":")[0]

	for _, allowedIP := range s.WhitelistAPIIP.IPs {
		if ip == allowedIP {
			log.Printf("[SECURITY] Ã¢ÂÂ Whitelisted API IP: %s", ip)
			return true
		}
	}

	log.Printf("[SECURITY] Ã¢ÂÂ IP not whitelisted for API: %s", ip)
	return false
}


func (s *SecuritySettings) ValidateCNCAccess(ip, clientVersion string) (bool, string) {
	
	if s.IsIPBlacklisted(ip) {
		return false, fmt.Sprintf("IP %s is blacklisted", ip)
	}

	
	if !s.IsIPWhitelistedForCNC(ip) {
		return false, fmt.Sprintf("IP %s is not whitelisted for CNC access", ip)
	}

	
	if valid, reason := s.ValidateSSHClient(clientVersion); !valid {
		return false, reason
	}

	return true, ""
}

func (s *SecuritySettings) AddBlockedSSHClient(client string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	
	for _, blocked := range s.BlockedSSHClients {
		if blocked == client {
			return fmt.Errorf("SSH client '%s' already in blocked list", client)
		}
	}

	s.BlockedSSHClients = append(s.BlockedSSHClients, client)
	log.Printf("[SECURITY] Added SSH client to blocklist: %s", client)
	return s.saveConfig()
}


func (s *SecuritySettings) RemoveBlockedSSHClient(client string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newList := []string{}
	found := false

	for _, blocked := range s.BlockedSSHClients {
		if blocked != client {
			newList = append(newList, blocked)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("SSH client '%s' not found in blocked list", client)
	}

	s.BlockedSSHClients = newList
	log.Printf("[SECURITY] Removed SSH client from blocklist: %s", client)
	return s.saveConfig()
}


func (s *SecuritySettings) AddIPToBlacklist(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ip = strings.Split(ip, ":")[0] 

	for _, blockedIP := range s.BlacklistIP.IPs {
		if blockedIP == ip {
			return fmt.Errorf("IP '%s' already in blacklist", ip)
		}
	}

	s.BlacklistIP.IPs = append(s.BlacklistIP.IPs, ip)
	log.Printf("[SECURITY] Added IP to blacklist: %s", ip)
	return s.saveConfig()
}


func (s *SecuritySettings) RemoveIPFromBlacklist(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ip = strings.Split(ip, ":")[0]
	newList := []string{}
	found := false

	for _, blockedIP := range s.BlacklistIP.IPs {
		if blockedIP != ip {
			newList = append(newList, blockedIP)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("IP '%s' not found in blacklist", ip)
	}

	s.BlacklistIP.IPs = newList
	log.Printf("[SECURITY] Removed IP from blacklist: %s", ip)
	return s.saveConfig()
}


func (s *SecuritySettings) saveConfig() error {
	
	data, err := os.ReadFile("assets/config.json")
	if err != nil {
		return fmt.Errorf("error reading config: %v", err)
	}

	var fullConfig ConfigWithSecurity
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	fullConfig.SecuritySettings = *s

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	if err := os.WriteFile("assets/config.json", updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config: %v", err)
	}

	return nil
}


func (s *SecuritySettings) GetSecurityStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"blocked_ssh_clients":       len(s.BlockedSSHClients),
		"whitelisted_ssh_clients":   len(s.WhitelistedSSHClients.List),
		"ssh_whitelist_enabled":     s.WhitelistedSSHClients.Enabled,
		"cnc_whitelisted_ips":       len(s.WhitelistCNCIP.IPs),
		"cnc_whitelist_enabled":     s.WhitelistCNCIP.Enabled,
		"api_whitelisted_ips":       len(s.WhitelistAPIIP.IPs),
		"api_whitelist_enabled":     s.WhitelistAPIIP.Enabled,
		"blacklisted_ips":           len(s.BlacklistIP.IPs),
		"blacklist_enabled":         s.BlacklistIP.Enabled,
	}
}