package cmds

import (
	"arismcnc/database"
	"fmt"
	"io"
	"strings"

	"github.com/gliderlabs/ssh"
)

type EchoCommand struct{}

func (c *EchoCommand) Name() string {
	return "echo"
}

func (c *EchoCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 1 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: echo [text]\033[0m")
		return
	}

	
	text := strings.Join(args, " ")
	fmt.Fprintln(output, text)
}

func (c *EchoCommand) Aliases() []string {
	return []string{"echo"}
}

func (c *EchoCommand) AdminOnly() bool {
	return false
}

func init() {
	CommandMap["echo"] = &EchoCommand{}
}
