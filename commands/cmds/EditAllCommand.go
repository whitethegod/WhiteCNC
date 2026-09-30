package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gliderlabs/ssh"
)

type EditAllCommand struct{}

func (c *EditAllCommand) Name() string {
	return "editall"
}

func (c *EditAllCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {

	if len(args) > 0 {
		daysStr := strings.TrimSpace(args[0])
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintln(output, "Invalid number of days.")
			return
		}

		if err := db.AddDaysEveryone(days); err != nil {
			fmt.Fprintf(output, "Failed to add days to users: %v\n", err)
			return
		}

		fmt.Fprintf(output, "Successfully added %d days to all users.\n", days)
		return
	}

	fmt.Fprintf(output, "How many days do you want to add to ALL users: ")

	daysInput, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading days input:", err)
		return
	}

	daysInput = strings.TrimSpace(daysInput)
	days, err := strconv.Atoi(daysInput)
	if err != nil {
		fmt.Fprintln(output, "Invalid number of days.")
		return
	}

	if err := db.AddDaysEveryone(days); err != nil {
		fmt.Fprintf(output, "Failed to add days to users: %v\n", err)
		return
	}

	fmt.Fprintf(output, "Successfully added %d days to all users.\n", days)
}

func (c *EditAllCommand) AdminOnly() bool {
	return true
}

func (c *EditAllCommand) Aliases() []string {
	return []string{"editall", "editsall"}
}

func init() {
	CommandMap["editall"] = &EditAllCommand{}
}
