package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"

	"github.com/gliderlabs/ssh"
)

type PasswordCommand struct{}

func (c *PasswordCommand) Name() string {
	return "password"
}

func (c *PasswordCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	
	fmt.Fprintf(output, "Enter current password (leave empty to cancel): ")
	currentPassword, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading input:", err)
		return
	}

	
	if currentPassword == "" {
		fmt.Fprintln(output, "\033[37;1mPassword change canceled.")
		return
	}

	
	isValid, err := db.VerifyPassword(session.User(), currentPassword)
	if err != nil {
		fmt.Fprintln(output, "\033[31;1mError verifying password:", err)
		return
	}

	if !isValid {
		fmt.Fprintln(output, "\033[31;1mInvalid current password. Please try again.")
		return
	}

	
	fmt.Fprintf(output, "Enter new password: ")
	password1, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading input:", err)
		return
	}

	
	fmt.Fprintf(output, "Confirm new password: ")
	password2, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading input:", err)
		return
	}

	
	if password1 != password2 {
		fmt.Fprintln(output, "\033[37;1mPasswords do not match. Please try again.")
		return
	}

	
	err = db.ChangePassword(session.User(), password1)
	if err != nil {
		fmt.Fprintln(output, "\033[37;1mError changing password:", err)
		return
	}

	fmt.Fprintln(output, "\033[37;1mPassword changed successfully!")
}

func (c *PasswordCommand) AdminOnly() bool {
	return false
}

func (c *PasswordCommand) Aliases() []string {
	return []string{"password", "passwd"}
}


func init() {
	CommandMap["password"] = &PasswordCommand{}
}
