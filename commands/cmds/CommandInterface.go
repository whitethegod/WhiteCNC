package cmds

import (
	"arismcnc/database"
	"io"

	"github.com/gliderlabs/ssh"
)

type Command interface {
	Name() string
	Execute(session ssh.Session, db *database.Database, args []string, output io.Writer)
	AdminOnly() bool
	Aliases() []string 
}


var CommandMap = map[string]Command{}
