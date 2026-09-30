package cmds

import (
	"arismcnc/database"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/gliderlabs/ssh"
)

type EncryptCommand struct{}

func (c *EncryptCommand) Name() string {
	return "encrypt"
}

func (c *EncryptCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 1 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: encrypt [text]\033[0m")
		return
	}

	
	text := strings.Join(args, " ")

	
	hasher := sha256.New()
	hasher.Write([]byte(text))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	
	fmt.Fprintf(output, "\033[92mOriginal text: %s\nSHA-256 hash: %s\033[0m\n", text, hash)
}

func (c *EncryptCommand) Aliases() []string {
	return []string{"encrypt", "sha256"}
}

func (c *EncryptCommand) AdminOnly() bool {
	return false
}

func init() {
	CommandMap["encrypt"] = &EncryptCommand{}
}
