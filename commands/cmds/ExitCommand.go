package cmds

import (
	"arismcnc/database"
	"fmt"
	"io"
	"os"

	"github.com/gliderlabs/ssh"
)


type ExitCommand struct{}

func (c *ExitCommand) Name() string {
	return "exit"
}

func (c *ExitCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	fmt.Fprintln(output, "")
	session.Close()
	os.Exit(0)
}

func (c *ExitCommand) AdminOnly() bool {
	return false
}


func (c *ExitCommand) Aliases() []string {
	return []string{"exit", "quit", "logout"}
}


func init() {
	CommandMap["exit"] = &ExitCommand{}
}
