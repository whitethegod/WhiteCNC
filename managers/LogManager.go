package managers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)





type LogConfig struct {
	Global               GlobalConfig              `json:"global"`
	Telegram             TelegramConfig            `json:"telegram"`
	Discord              DiscordConfig             `json:"discord"`
	DiscordIntegration   DiscordIntegrationConfig  `json:"Discord_Integration"`
	TelegramIntegration  TelegramIntegrationConfig `json:"Telegram_Integration"`
}

type GlobalConfig struct {
	Enabled    bool `json:"enabled"`
	LogInFiles bool `json:"log_in_files"`
}

type TelegramConfig struct {
	Enabled  bool     `json:"enabled"`
	BotToken string   `json:"bot_token"`
	ChatIDs  []string `json:"chat_ids"`
}

type DiscordConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
}


type DiscordIntegrationConfig struct {
	AttackSent               EventWebhookConfig `json:"Attack_Sent"`
	BlacklistedHost          EventWebhookConfig `json:"Blacklisted_Host"`
	AdminActions             EventWebhookConfig `json:"Admin_Actions"`
	AccountCreationRegister  EventWebhookConfig `json:"Account_Creation_Register"`
	APILogs                  EventWebhookConfig `json:"API_Logs"`
	UserLogin                EventWebhookConfig `json:"User_Login"`
}

type EventWebhookConfig struct {
	WebhookURL string `json:"webhook_url"`
	Title      string `json:"title"`
	Image      string `json:"image"`
	HexColor   string `json:"hex_color"`
	Enabled    bool   `json:"enabled"`
}


type TelegramIntegrationConfig struct {
	AttackSent               TelegramEventConfig `json:"Attack_Sent"`
	BlacklistedHost          TelegramEventConfig `json:"Blacklisted_Host"`
	AdminActions             TelegramEventConfig `json:"Admin_Actions"`
	AccountCreationRegister  TelegramEventConfig `json:"Account_Creation_Register"`
	APILogs                  TelegramEventConfig `json:"API_Logs"`
	ReportLogs               TelegramEventConfig `json:"Report_Logs"`
	UserLogin                TelegramEventConfig `json:"User_Login"`
}

type TelegramEventConfig struct {
	Token   string `json:"token"`
	ChatID  string `json:"chat_id"`
	Title   string `json:"title"`
	Enabled bool   `json:"enabled"`
}





type LogManager struct {
	config  LogConfig
	logFile *os.File
}


type EventType string

const (
	EventAttackSent      EventType = "Attack_Sent"
	EventBlacklisted     EventType = "Blacklisted_Host"
	EventAdminAction     EventType = "Admin_Actions"
	EventAccountCreation EventType = "Account_Creation_Register"
	EventAPILog          EventType = "API_Logs"
	EventReportLog       EventType = "Report_Logs"
	EventUserLogin       EventType = "User_Login"
)





func NewLogManager(configPath string) (*LogManager, error) {
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config LogConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var logFile *os.File
	if config.Global.Enabled && config.Global.LogInFiles {
		logFile, err = os.OpenFile("./assets/logs/global_logs.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	return &LogManager{config: config, logFile: logFile}, nil
}






func (lm *LogManager) LogEvent(eventType EventType, data map[string]interface{}) {
	log.Printf("[LogManager] LogEvent called with type: %s", eventType)
	
	if !lm.config.Global.Enabled {
		log.Printf("[LogManager] Global logging is disabled")
		return
	}

	message := lm.formatEventMessage(eventType, data)
	log.Printf("[LogManager] Formatted message: %s", message)

	if lm.config.Global.LogInFiles && lm.logFile != nil {
		log.Printf("[LogManager] Writing to file...")
		lm.writeToFile(message)
	}

	log.Printf("[LogManager] Sending to Discord...")
	lm.sendEventToDiscord(eventType, data)

	log.Printf("[LogManager] Sending to Telegram...")
	lm.sendEventToTelegram(eventType, data)
	
	log.Printf("[LogManager] LogEvent completed")
}


func (lm *LogManager) Log(message string) {
	if !lm.config.Global.Enabled {
		return
	}

	
	if lm.config.Global.LogInFiles && lm.logFile != nil {
		lm.writeToFile(message)
	}

	
	if lm.config.Telegram.Enabled {
		lm.sendToTelegram(message)
	}

	
	if lm.config.Discord.Enabled {
		lm.sendToDiscord(message)
	}
}






func (lm *LogManager) LogAttackSent(username, target, port, duration, method string) {
	lm.LogEvent(EventAttackSent, map[string]interface{}{
		"username": username,
		"target":   target,
		"port":     port,
		"duration": duration,
		"method":   method,
	})
}


func (lm *LogManager) LogBlacklistedAttempt(username, target, port, duration, method string) {
	lm.LogEvent(EventBlacklisted, map[string]interface{}{
		"username": username,
		"target":   target,
		"port":     port,
		"duration": duration,
		"method":   method,
	})
}


func (lm *LogManager) LogAdminAction(admin, action, target, details string) {
	lm.LogEvent(EventAdminAction, map[string]interface{}{
		"admin":   admin,
		"action":  action,
		"target":  target,
		"details": details,
	})
}


func (lm *LogManager) LogAccountCreation(createdBy, username string, concurrents, maxtime, days, admin, vip, apiAccess, powerSaving, spamBypass, blacklistBypass int) {
	lm.LogEvent(EventAccountCreation, map[string]interface{}{
		"created_by":        createdBy,
		"username":          username,
		"concurrents":       concurrents,
		"maxtime":           maxtime,
		"days":              days,
		"admin":             admin,
		"vip":               vip,
		"api_access":        apiAccess,
		"power_saving":      powerSaving,
		"spam_bypass":       spamBypass,
		"blacklist_bypass":  blacklistBypass,
	})
}


func (lm *LogManager) LogUserLogin(username, ip, sshClient, country, region, city, isp, asn string, admin, vip, private int) {
	lm.LogEvent(EventUserLogin, map[string]interface{}{
		"username":   username,
		"ip":         ip,
		"ssh_client": sshClient,
		"country":    country,
		"region":     region,
		"city":       city,
		"isp":        isp,
		"asn":        asn,
		"admin":      admin,
		"vip":        vip,
		"private":    private,
	})
}


func (lm *LogManager) LogAPIAccess(username, endpoint, method, status string) {
	lm.LogEvent(EventAPILog, map[string]interface{}{
		"username": username,
		"endpoint": endpoint,
		"method":   method,
		"status":   status,
	})
}


func (lm *LogManager) LogReport(username, reportType, details string) {
	lm.LogEvent(EventReportLog, map[string]interface{}{
		"username":    username,
		"report_type": reportType,
		"details":     details,
	})
}





func (lm *LogManager) formatEventMessage(eventType EventType, data map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	switch eventType {
	case EventAttackSent:
		return fmt.Sprintf("[%s] ATTACK SENT | User: %s | Target: %s | Port: %s | Duration: %ss | Method: %s",
			timestamp, data["username"], data["target"], data["port"], data["duration"], data["method"])
			
	case EventBlacklisted:
		return fmt.Sprintf("[%s] BLACKLIST ATTEMPT | User: %s | Target: %s | Port: %s | Duration: %ss | Method: %s",
			timestamp, data["username"], data["target"], data["port"], data["duration"], data["method"])
			
	case EventAdminAction:
		return fmt.Sprintf("[%s] ADMIN ACTION | Admin: %s | Action: %s | Target: %s | Details: %s",
			timestamp, data["admin"], data["action"], data["target"], data["details"])
			
	case EventAccountCreation:
		return fmt.Sprintf("[%s] ACCOUNT CREATED | Created by: %s | Username: %s | Plan: %s",
			timestamp, data["created_by"], data["username"], data["plan"])
			
	case EventUserLogin:
		return fmt.Sprintf("[%s] USER LOGIN | Username: %s | IP: %s | Client: %s | Location: %s, %s | ISP: %s",
			timestamp, data["username"], data["ip"], data["ssh_client"], data["city"], data["country"], data["isp"])
			
	case EventAPILog:
		return fmt.Sprintf("[%s] API ACCESS | User: %s | Endpoint: %s | Method: %s | Status: %s",
			timestamp, data["username"], data["endpoint"], data["method"], data["status"])
			
	case EventReportLog:
		return fmt.Sprintf("[%s] REPORT | User: %s | Type: %s | Details: %s",
			timestamp, data["username"], data["report_type"], data["details"])
			
	default:
		return fmt.Sprintf("[%s] UNKNOWN EVENT: %v", timestamp, data)
	}
}





func (lm *LogManager) sendEventToDiscord(eventType EventType, data map[string]interface{}) {
	var webhookConfig EventWebhookConfig
	var enabled bool

	
	switch eventType {
	case EventAttackSent:
		webhookConfig = lm.config.DiscordIntegration.AttackSent
		enabled = webhookConfig.Enabled
	case EventBlacklisted:
		webhookConfig = lm.config.DiscordIntegration.BlacklistedHost
		enabled = webhookConfig.Enabled
	case EventAdminAction:
		webhookConfig = lm.config.DiscordIntegration.AdminActions
		enabled = webhookConfig.Enabled
	case EventAccountCreation:
		webhookConfig = lm.config.DiscordIntegration.AccountCreationRegister
		enabled = webhookConfig.Enabled
	case EventAPILog:
		webhookConfig = lm.config.DiscordIntegration.APILogs
		enabled = webhookConfig.Enabled
	case EventUserLogin:
		webhookConfig = lm.config.DiscordIntegration.UserLogin
		enabled = webhookConfig.Enabled
	default:
		return 
	}

	if !enabled || webhookConfig.WebhookURL == "" {
		return
	}

	
	embed := lm.createDiscordEmbed(webhookConfig, eventType, data)

	payload := map[string]interface{}{
		"embeds": []interface{}{embed},
	}

	jsonData, _ := json.Marshal(payload)
	
	resp, err := http.Post(webhookConfig.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("failed to send Discord webhook (%s): %v", eventType, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Discord webhook error (%s): status %d, body: %s", eventType, resp.StatusCode, string(body))
	}
}

func (lm *LogManager) createDiscordEmbed(config EventWebhookConfig, eventType EventType, data map[string]interface{}) map[string]interface{} {
	
	colorHex := config.HexColor
	if len(colorHex) > 0 && colorHex[0] == '#' {
		colorHex = colorHex[1:]
	}
	var colorInt int
	fmt.Sscanf(colorHex, "%x", &colorInt)

	fields := lm.createEmbedFields(eventType, data)

	embed := map[string]interface{}{
		"title":       config.Title,
		"color":       colorInt,
		"fields":      fields,
		"timestamp":   time.Now().Format(time.RFC3339),
		"footer": map[string]string{
			"text": "C2 Logging System",
		},
	}

	
	if config.Image != "" {
		embed["thumbnail"] = map[string]string{
			"url": config.Image,
		}
	}

	return embed
}

func boolToEmoji(value interface{}) string {
	intVal, ok := value.(int)
	if ok {
		if intVal == 1 {
			return "\u2705 True"
		}
		return "\u274C False"
	}
	
	boolVal, ok := value.(bool)
	if ok {
		if boolVal {
			return "\u2705 True"
		}
		return "\u274C False"
	}
	
	return "\u274C False"
}

func boolToEmojiTelegram(value interface{}) string {
	intVal, ok := value.(int)
	if ok {
		if intVal == 1 {
			return "\u2705 True"
		}
		return "\u274C False"
	}
	
	boolVal, ok := value.(bool)
	if ok {
		if boolVal {
			return "\u2705 True"
		}
		return "\u274C False"
	}
	
	return "\u274C False"
}

func (lm *LogManager) createEmbedFields(eventType EventType, data map[string]interface{}) []map[string]interface{} {
	var fields []map[string]interface{}

	switch eventType {
	case EventAttackSent:
		fields = []map[string]interface{}{
			{"name": "?? Username", "value": data["username"], "inline": true},
			{"name": "?? Target", "value": data["target"], "inline": true},
			{"name": "?? Port", "value": data["port"], "inline": true},
			{"name": "?? Duration", "value": fmt.Sprintf("%ss", data["duration"]), "inline": true},
			{"name": "? Method", "value": data["method"], "inline": true},
			{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
		}
		
	case EventBlacklisted:
		fields = []map[string]interface{}{
			{"name": "?? Username", "value": data["username"], "inline": true},
			{"name": "?? Target (Blocked)", "value": data["target"], "inline": true},
			{"name": "?? Port", "value": data["port"], "inline": true},
			{"name": "?? Duration", "value": fmt.Sprintf("%ss", data["duration"]), "inline": true},
			{"name": "? Method", "value": data["method"], "inline": true},
			{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
		}
		
	case EventAdminAction:
		fields = []map[string]interface{}{
			{"name": "?? Admin", "value": data["admin"], "inline": true},
			{"name": "??? Action", "value": data["action"], "inline": true},
			{"name": "?? Target", "value": data["target"], "inline": true},
			{"name": "?? Details", "value": data["details"], "inline": false},
			{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
		}
		
case EventAccountCreation:
	// Converter valores para string de forma segura
	concurrents := fmt.Sprintf("%v", data["concurrents"])
	maxtime := fmt.Sprintf("%v", data["maxtime"])
	days := fmt.Sprintf("%v", data["days"])
	
	fields = []map[string]interface{}{
		{"name": "?? Created By", "value": fmt.Sprintf("%v", data["created_by"]), "inline": true},
		{"name": "?? Username", "value": fmt.Sprintf("%v", data["username"]), "inline": true},
		{"name": "?? Concurrents", "value": concurrents, "inline": true},
		{"name": "?? MaxTime", "value": maxtime + "s", "inline": true},
		{"name": "?? Plan Days", "value": days + " days", "inline": true},
		{"name": "?? Admin", "value": boolToEmoji(data["admin"]), "inline": true},
		{"name": "? VIP", "value": boolToEmoji(data["vip"]), "inline": true},
		{"name": "?? API Access", "value": boolToEmoji(data["api_access"]), "inline": true},
		{"name": "? Power Saving Bypass", "value": boolToEmoji(data["power_saving"]), "inline": true},
		{"name": "??? Spam Bypass", "value": boolToEmoji(data["spam_bypass"]), "inline": true},
		{"name": "?? Blacklist Bypass", "value": boolToEmoji(data["blacklist_bypass"]), "inline": true},
		{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
	}
		
	case EventAPILog:
		fields = []map[string]interface{}{
			{"name": "?? Username", "value": data["username"], "inline": true},
			{"name": "?? Endpoint", "value": data["endpoint"], "inline": true},
			{"name": "?? Method", "value": data["method"], "inline": true},
			{"name": "? Status", "value": data["status"], "inline": true},
			{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
		}
		
	case EventUserLogin:
		fields = []map[string]interface{}{
			{"name": "?? Username", "value": data["username"], "inline": true},
			{"name": "?? IP Address", "value": data["ip"], "inline": true},
			{"name": "?? SSH Client", "value": data["ssh_client"], "inline": true},
			{"name": "?? Country", "value": data["country"], "inline": true},
			{"name": "?? Region", "value": data["region"], "inline": true},
			{"name": "??? City", "value": data["city"], "inline": true},
			{"name": "?? ISP", "value": data["isp"], "inline": true},
			{"name": "?? ASN", "value": data["asn"], "inline": true},
			{"name": "?? Admin", "value": boolToEmoji(data["admin"]), "inline": true},
			{"name": "? VIP", "value": boolToEmoji(data["vip"]), "inline": true},
			{"name": "?? Private", "value": boolToEmoji(data["private"]), "inline": true},
			{"name": "?? Time", "value": time.Now().Format("15:04:05"), "inline": true},
		}
	}

	return fields
}

func (lm *LogManager) sendEventToTelegram(eventType EventType, data map[string]interface{}) {
	var telegramConfig TelegramEventConfig
	var enabled bool

	
	switch eventType {
	case EventAttackSent:
		telegramConfig = lm.config.TelegramIntegration.AttackSent
		enabled = telegramConfig.Enabled
	case EventBlacklisted:
		telegramConfig = lm.config.TelegramIntegration.BlacklistedHost
		enabled = telegramConfig.Enabled
	case EventAdminAction:
		telegramConfig = lm.config.TelegramIntegration.AdminActions
		enabled = telegramConfig.Enabled
	case EventAccountCreation:
		telegramConfig = lm.config.TelegramIntegration.AccountCreationRegister
		enabled = telegramConfig.Enabled
	case EventAPILog:
		telegramConfig = lm.config.TelegramIntegration.APILogs
		enabled = telegramConfig.Enabled
	case EventReportLog:
		telegramConfig = lm.config.TelegramIntegration.ReportLogs
		enabled = telegramConfig.Enabled
	case EventUserLogin:
		telegramConfig = lm.config.TelegramIntegration.UserLogin
		enabled = telegramConfig.Enabled
	default:
		return
	}

	if !enabled || telegramConfig.Token == "" || telegramConfig.ChatID == "" {
		return
	}

	
	if telegramConfig.Token == "https://t.me/botfather for Create Bot Telegram" {
		return
	}

	message := lm.formatTelegramMessage(telegramConfig.Title, eventType, data)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramConfig.Token)
	payload := map[string]interface{}{
		"chat_id":    telegramConfig.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("failed to send Telegram message (%s): %v", eventType, err)
		return
	}
	defer resp.Body.Close()
}

func (lm *LogManager) formatTelegramMessage(title string, eventType EventType, data map[string]interface{}) string {
    timestamp := time.Now().Format("2006-01-02 15:04:05")

    // Titulo do evento
    message := fmt.Sprintf("<b>\U0001F514 %s</b>\n\n", title)

    switch eventType {

    case EventAdminAction:
        message += fmt.Sprintf("\U0001F451 <b>Admin:</b> %s\n", data["admin"])
        message += fmt.Sprintf("\U0001F6E0 <b>Action:</b> %s\n", data["action"])
        message += fmt.Sprintf("\U0001F3AF <b>Target:</b> %s\n", data["target"])
        message += fmt.Sprintf("\U0001F4C4 <b>Details:</b> %s\n", data["details"])

    case EventAccountCreation:
        message += fmt.Sprintf("\U0001F468 <b>Created By:</b> %s\n", data["created_by"])
        message += fmt.Sprintf("\U0001F464 <b>Username:</b> %s\n", data["username"])
        message += fmt.Sprintf("\U0001F522 <b>Concurrents:</b> %v\n", data["concurrents"])
        message += fmt.Sprintf("\u23F1 <b>MaxTime:</b> %vs\n", data["maxtime"])
        message += fmt.Sprintf("\U0001F4C5 <b>Plan Days:</b> %v days\n", data["days"])
        message += fmt.Sprintf("\U0001F451 <b>Admin:</b> %s\n", boolToEmojiTelegram(data["admin"]))
        message += fmt.Sprintf("\u2B50 <b>VIP:</b> %s\n", boolToEmojiTelegram(data["vip"]))
        message += fmt.Sprintf("\U0001F511 <b>API Access:</b> %s\n", boolToEmojiTelegram(data["api_access"]))
        message += fmt.Sprintf("\u26A1 <b>Power Saving Bypass:</b> %s\n", boolToEmojiTelegram(data["power_saving"]))
        message += fmt.Sprintf("\U0001F6E1 <b>Spam Bypass:</b> %s\n", boolToEmojiTelegram(data["spam_bypass"]))
        message += fmt.Sprintf("\U0001F6AB <b>Blacklist Bypass:</b> %s\n", boolToEmojiTelegram(data["blacklist_bypass"]))

    case EventAPILog:
        message += fmt.Sprintf("\U0001F464 <b>User:</b> %s\n", data["username"])
        message += fmt.Sprintf("\U0001F4CD <b>Endpoint:</b> <code>%s</code>\n", data["endpoint"])
        message += fmt.Sprintf("\U0001F4EC <b>Method:</b> %s\n", data["method"])
        message += fmt.Sprintf("\u2714 <b>Status:</b> %s\n", data["status"])

    case EventReportLog:
        message += fmt.Sprintf("\U0001F464 <b>User:</b> %s\n", data["username"])
        message += fmt.Sprintf("\U0001F4CC <b>Type:</b> %s\n", data["report_type"])
        message += fmt.Sprintf("\U0001F4C4 <b>Details:</b> %s\n", data["details"])

    case EventUserLogin:
        message += fmt.Sprintf("\U0001F464 <b>Username:</b> %s\n", data["username"])
        message += fmt.Sprintf("\U0001F310 <b>IP Address:</b> <code>%s</code>\n", data["ip"])
        message += fmt.Sprintf("\U0001F4BB <b>SSH Client:</b> <code>%s</code>\n", data["ssh_client"])
        message += fmt.Sprintf("\U0001F30D <b>Country:</b> %s\n", data["country"])
        message += fmt.Sprintf("\U0001F4CD <b>Region:</b> %s\n", data["region"])
        message += fmt.Sprintf("\U0001F3D9 <b>City:</b> %s\n", data["city"])
        message += fmt.Sprintf("\U0001F50C <b>ISP:</b> %s\n", data["isp"])
        message += fmt.Sprintf("\U0001F522 <b>ASN:</b> %s\n", data["asn"])
        message += fmt.Sprintf("\U0001F451 <b>Admin:</b> %s\n", boolToEmojiTelegram(data["admin"]))
        message += fmt.Sprintf("\u2B50 <b>VIP:</b> %s\n", boolToEmojiTelegram(data["vip"]))
        message += fmt.Sprintf("\U0001F512 <b>Private:</b> %s\n", boolToEmojiTelegram(data["private"]))

    }

    message += fmt.Sprintf("\n\U0001F552 <b>Time:</b> %s", timestamp)
    return message
}

func (lm *LogManager) sendToTelegram(message string) {
	for _, chatID := range lm.config.Telegram.ChatIDs {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", lm.config.Telegram.BotToken)
		payload := map[string]interface{}{
			"chat_id": chatID,
			"text":    message,
		}
		data, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("failed to send Telegram message (%s): %v", chatID, err)
			continue
		}
		resp.Body.Close()
	}
}

func (lm *LogManager) sendToDiscord(message string) {
	payload := map[string]interface{}{
		"content": message,
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(lm.config.Discord.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("failed to send Discord message: %v", err)
		return
	}
	resp.Body.Close()
}





func (lm *LogManager) writeToFile(message string) {
	if lm.logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		lm.logFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
	}
}

func (lm *LogManager) Close() {
	if lm.logFile != nil {
		lm.logFile.Close()
	}
}