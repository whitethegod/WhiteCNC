package cmds

import (
	"arismcnc/database"
	"fmt"
	"io"

	"github.com/gliderlabs/ssh"
)

type CreditsCommand struct{}

func (c *CreditsCommand) Name() string {
	return "credits"
}

func (c *CreditsCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	fmt.Fprintln(output, "NeverCNC - Ver. Legacy")
	fmt.Fprintln(output, "Contact: @WhiteCode")
}

func (c *CreditsCommand) AdminOnly() bool {
	return false
}

func (c *CreditsCommand) Aliases() []string {
	return []string{"creds", "credits"}
}

func init() {
	CommandMap["credits"] = &CreditsCommand{}
}
