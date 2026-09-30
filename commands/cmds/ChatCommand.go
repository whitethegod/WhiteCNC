package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
)


type ChatMessage struct {
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Role      string    `json:"role"` 
}


type ChatSettings struct {
	Enabled                     bool   `json:"Enabled"`
	Title                       string `json:"Title"`
	MaxMessages                 int    `json:"Max_Messages"`
	MaxUsers                    int    `json:"Max_Users"`
	MaxMessagesPerUserPerMinute int    `json:"Max_Messages_Per_User_Per_Minute"`
	Restrictions                struct {
		OnlyAdminsCanAccessChat    bool `json:"Only_Admins_Can_Access_Chat"`
		OnlyResellersCanAccessChat bool `json:"Only_Resellers_Can_Access_Chat"`
		OnlyVIPsCanAccessChat      bool `json:"Only_VIPs_Can_Access_Chat"`
		OnlyHoldersCanAccessChat   bool `json:"Only_Holders_Can_Access_Chat"`
		OnlyUsersCanAccessChat     bool `json:"Only_Users_Can_Access_Chat"`
	} `json:"Restrictions"`
}


type BroadcastFunc func(message string, isChatMessage bool)


type ChatManager struct {
	messages       []ChatMessage
	userRateLimit  map[string][]time.Time 
	settings       ChatSettings
	broadcastFunc  BroadcastFunc
	mu             sync.RWMutex
	activeChatUsers map[string]bool 
}

var globalChatManager *ChatManager
var chatInitOnce sync.Once


func GetChatManager() *ChatManager {
	chatInitOnce.Do(func() {
		globalChatManager = &ChatManager{
			messages:         make([]ChatMessage, 0),
			userRateLimit:    make(map[string][]time.Time),
			activeChatUsers:  make(map[string]bool),
		}
		globalChatManager.LoadSettings()
	})
	return globalChatManager
}


func (cm *ChatManager) SetBroadcastFunc(fn BroadcastFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.broadcastFunc = fn
}


func (cm *ChatManager) LoadSettings() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	file, err := os.Open("assets/config.json")
	if err != nil {
		log.Printf("[CHAT] Error opening config.json: %v", err)
		return err
	}
	defer file.Close()

	var config map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Printf("[CHAT] Error decoding config.json: %v", err)
		return err
	}

	
	if chatSettings, ok := config["Chat_Settings"].(map[string]interface{}); ok {
		cm.settings.Enabled = getBool(chatSettings, "Enabled", true)
		cm.settings.Title = getString(chatSettings, "Title", "=== GLOBAL CHAT ===")
		cm.settings.MaxMessages = getInt(chatSettings, "Max_Messages", 10000)
		cm.settings.MaxUsers = getInt(chatSettings, "Max_Users", 100)
		cm.settings.MaxMessagesPerUserPerMinute = getInt(chatSettings, "Max_Messages_Per_User_Per_Minute", 10)

		if restrictions, ok := chatSettings["Restrictions"].(map[string]interface{}); ok {
			cm.settings.Restrictions.OnlyAdminsCanAccessChat = getBool(restrictions, "Only_Admins_Can_Access_Chat", false)
			cm.settings.Restrictions.OnlyResellersCanAccessChat = getBool(restrictions, "Only_Resellers_Can_Access_Chat", false)
			cm.settings.Restrictions.OnlyVIPsCanAccessChat = getBool(restrictions, "Only_VIPs_Can_Access_Chat", false)
			cm.settings.Restrictions.OnlyHoldersCanAccessChat = getBool(restrictions, "Only_Holders_Can_Access_Chat", false)
			cm.settings.Restrictions.OnlyUsersCanAccessChat = getBool(restrictions, "Only_Users_Can_Access_Chat", false)
		}
	} else {
		
		cm.settings.Enabled = true
		cm.settings.Title = "=== GLOBAL CHAT ==="
		cm.settings.MaxMessages = 10000
		cm.settings.MaxUsers = 100
		cm.settings.MaxMessagesPerUserPerMinute = 10
	}

	return nil
}


func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getString(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}


var GetOnlineUsersCount = func() int {
	return 0
}


func (cm *ChatManager) CanAccessChat(userInfo database.AccountInfo) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	restrictions := cm.settings.Restrictions

	
	if !restrictions.OnlyAdminsCanAccessChat &&
		!restrictions.OnlyResellersCanAccessChat &&
		!restrictions.OnlyVIPsCanAccessChat &&
		!restrictions.OnlyHoldersCanAccessChat &&
		!restrictions.OnlyUsersCanAccessChat {
		return true
	}

	
	if restrictions.OnlyAdminsCanAccessChat && userInfo.Admin == 1 {
		return true
	}

	if restrictions.OnlyVIPsCanAccessChat && userInfo.Vip == 1 {
		return true
	}

	if restrictions.OnlyUsersCanAccessChat {
		return true
	}

	return false
}


func (cm *ChatManager) CheckRateLimit(username string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	
	if timestamps, exists := cm.userRateLimit[username]; exists {
		validTimestamps := []time.Time{}
		for _, ts := range timestamps {
			if ts.After(oneMinuteAgo) {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		cm.userRateLimit[username] = validTimestamps

		
		if len(validTimestamps) >= cm.settings.MaxMessagesPerUserPerMinute {
			return false
		}
	}

	return true
}


func (cm *ChatManager) AddMessage(username, message string, role string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	
	now := time.Now()
	cm.userRateLimit[username] = append(cm.userRateLimit[username], now)

	
	chatMsg := ChatMessage{
		Username:  username,
		Message:   message,
		Timestamp: now,
		Role:      role,
	}

	
	cm.messages = append(cm.messages, chatMsg)

	
	if len(cm.messages) > cm.settings.MaxMessages {
		cm.messages = cm.messages[1:]
	}

	return nil
}


func (cm *ChatManager) GetMessages(count int) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if count > len(cm.messages) {
		count = len(cm.messages)
	}

	if count == 0 {
		return cm.messages
	}

	start := len(cm.messages) - count
	if start < 0 {
		start = 0
	}

	return cm.messages[start:]
}


func (cm *ChatManager) ClearMessages() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.messages = make([]ChatMessage, 0)
}


func (cm *ChatManager) GetTotalMessages() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.messages)
}


func (cm *ChatManager) GetActiveUsersCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.userRateLimit)
}


func (cm *ChatManager) AddUserToChat(username string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeChatUsers[username] = true
}


func (cm *ChatManager) RemoveUserFromChat(username string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.activeChatUsers, username)
}


func (cm *ChatManager) IsUserInChat(username string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.activeChatUsers[username]
}


func (cm *ChatManager) BroadcastMessage(msg ChatMessage) {
	cm.mu.RLock()
	broadcastFunc := cm.broadcastFunc
	cm.mu.RUnlock()

	if broadcastFunc == nil {
		return
	}

	
	timestamp := msg.Timestamp.Format("15:04:05")
	formattedMsg := fmt.Sprintf("[%s] [%s] %s: %s", 
		timestamp, strings.ToUpper(msg.Role), msg.Username, msg.Message)

	
	broadcastFunc(formattedMsg, true)
}


func (cm *ChatManager) BroadcastSystemMessage(message string) {
	cm.mu.RLock()
	broadcastFunc := cm.broadcastFunc
	cm.mu.RUnlock()

	if broadcastFunc == nil {
		return
	}

	formattedMsg := fmt.Sprintf("[SYSTEM] %s", message)
	
	broadcastFunc(formattedMsg, false)
}


func getUserRole(userInfo database.AccountInfo) string {
	if userInfo.Admin == 1 {
		return "admin"
	}
	if userInfo.Vip == 1 {
		return "vip"
	}
	return "user"
}


type ChatCommand struct{}

func (c *ChatCommand) Name() string {
	return "chat"
}

func (c *ChatCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	chatManager := GetChatManager()
	userInfo := db.GetAccountInfo(session.User())

	
	if !chatManager.settings.Enabled {
		fmt.Fprintln(output, "[!] The chat system is currently disabled.")
		return
	}

	
	if !chatManager.CanAccessChat(userInfo) {
		fmt.Fprintln(output, "[!] You don't have permission to access the chat.")
		return
	}

	
	onlineUsers := GetOnlineUsersCount()
	if onlineUsers > chatManager.settings.MaxUsers {
		fmt.Fprintln(output, "[!] Chat is full. Maximum users reached.")
		return
	}

	
	c.enterChat(session, db, output, chatManager)
}

func (c *ChatCommand) enterChat(session ssh.Session, db *database.Database, output io.Writer, cm *ChatManager) {
	userInfo := db.GetAccountInfo(session.User())
	username := session.User()
	role := getUserRole(userInfo)

	
	cm.AddUserToChat(username)
	defer cm.RemoveUserFromChat(username) 

	
	fmt.Fprintln(output, "=== ENTERING GLOBAL CHAT ===")
	fmt.Fprintln(output, "Type 'exit' to leave, '/help' for commands")
	fmt.Fprintln(output, "")

	
	messages := cm.GetMessages(10)
	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		fmt.Fprintf(output, "[%s] [%s] %s: %s\n", 
			timestamp, strings.ToUpper(msg.Role), msg.Username, msg.Message)
	}

	fmt.Fprintf(output, "\n[+] You joined the chat as %s\n", role)
	fmt.Fprintln(output, "")

	
	joinMsg := ChatMessage{
		Username:  "SYSTEM",
		Message:   fmt.Sprintf("%s joined the chat", username),
		Timestamp: time.Now(),
		Role:      "system",
	}
	cm.BroadcastMessage(joinMsg)

	
	for {
		
		fmt.Fprintf(output, "%s@botnet Ã¢ÂÂºÃ¢ÂÂº ", username)
		
		
		input, err := utils.ReadLine(session)
		if err != nil {
			break
		}
		
		input = strings.TrimSpace(input)
		
		
		if strings.HasPrefix(input, "/") || strings.ToLower(input) == "exit" {
			switch strings.ToLower(input) {
			case "exit", "/exit", "/quit", "/leave":
				
				leaveMsg := ChatMessage{
					Username:  "SYSTEM",
					Message:   fmt.Sprintf("%s left the chat", username),
					Timestamp: time.Now(),
					Role:      "system",
				}
				cm.BroadcastMessage(leaveMsg)
				fmt.Fprintln(output, "[!] Left the chat.")
				return
				
			case "/help", "/commands":
				c.showChatHelp(output)
				continue
				
			case "/clear":
				
				fmt.Fprint(output, "\033[2J\033[H")
				fmt.Fprintln(output, "=== GLOBAL CHAT ===")
				fmt.Fprintln(output, "")
				continue
				
			case "/users", "/online":
				onlineUsers := GetOnlineUsersCount()
				fmt.Fprintf(output, "Online users: %d\n", onlineUsers)
				continue
				
			default:
				fmt.Fprintln(output, "[!] Unknown command. Type /help for available commands.")
				continue
			}
		}

		
		if input == "" {
			continue
		}

		if len(input) > 500 {
			fmt.Fprintln(output, "[!] Message is too long. Maximum 500 characters.")
			continue
		}

		
		if !cm.CheckRateLimit(username) {
			fmt.Fprintf(output, "[!] Rate limit exceeded. Max %d messages per minute.\n", 
				cm.settings.MaxMessagesPerUserPerMinute)
			continue
		}

		
		if err := cm.AddMessage(username, input, role); err != nil {
			fmt.Fprintln(output, "[!] Failed to send message.")
			log.Printf("[CHAT] Error adding message: %v", err)
			continue
		}

		
		chatMsg := ChatMessage{
			Username:  username,
			Message:   input,
			Timestamp: time.Now(),
			Role:      role,
		}
		cm.BroadcastMessage(chatMsg)
	}
}

func (c *ChatCommand) showChatHelp(output io.Writer) {
	help := `
Chat Commands:

/help       - Show this help
exit       - Leave the chat
/clear      - Clear your screen
/users      - Show online users
<message>  - Send a message

Just type and press Enter to send messages!
`
	fmt.Fprintln(output, help)
}

func (c *ChatCommand) AdminOnly() bool {
	return false
}

func (c *ChatCommand) Aliases() []string {
	return []string{"c", "talk", "global"}
}


type BroadcastCommand struct{}

func (c *BroadcastCommand) Name() string {
	return "broadcast"
}

func (c *BroadcastCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	chatManager := GetChatManager()
	userInfo := db.GetAccountInfo(session.User())

	
	if userInfo.Admin != 1 {
		fmt.Fprintln(output, "[!] Only admins can use broadcast.")
		return
	}

	
	if len(args) == 0 {
		fmt.Fprint(output, "Write the message you want to send to all users: ")
		
		
		message, err := utils.ReadLine(session)
		if err != nil {
			fmt.Fprintln(output, "[!] Error reading message.")
			return
		}
		
		message = strings.TrimSpace(message)
		if message == "" {
			fmt.Fprintln(output, "[!] Message cannot be empty.")
			return
		}
		
		
		chatManager.BroadcastSystemMessage(message)
		fmt.Fprintln(output, "[Ã¢ÂÂ] Broadcast sent successfully to all users.")
		return
	}

	
	message := strings.Join(args, " ")
	chatManager.BroadcastSystemMessage(message)
	fmt.Fprintln(output, "[Ã¢ÂÂ] Broadcast sent successfully to all users.")
}

func (c *BroadcastCommand) AdminOnly() bool {
	return true
}

func (c *BroadcastCommand) Aliases() []string {
	return []string{"bc", "announce"}
}


func init() {
	CommandMap["chat"] = &ChatCommand{}
	CommandMap["broadcast"] = &BroadcastCommand{}
}