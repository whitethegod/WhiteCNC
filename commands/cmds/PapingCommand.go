package cmds

import (
	"arismcnc/database"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/gliderlabs/ssh"
)

type PapingCommand struct{}

func (p *PapingCommand) Name() string {
	return "paping"
}

func (p *PapingCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 2 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: paping [ip] [port] [-t count]\033[0m")
		return
	}

	address := args[0]
	port := args[1]
	timeout := 2 * time.Second

	
	if _, err := strconv.Atoi(port); err != nil {
		fmt.Fprintln(output, "\033[91mInvalid port. Please provide a valid numeric port.\033[0m")
		return
	}

	
	continuous := false
	count := 0
	if len(args) >= 4 && args[2] == "-t" {
		continuous = true
		var err error
		count, err = strconv.Atoi(args[3])
		if err != nil || count < 1 || count > 15 {
			fmt.Fprintln(output, "\033[91mInvalid count value. It must be a number between 1 and 15.\033[0m")
			return
		}
	}

	
	ping := func() {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, port), timeout)
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(output, "\033[91mError pinging %s:%s - %v\033[0m\n", address, port, err)
		} else {
			conn.Close()
			fmt.Fprintf(output, "\033[92m%s:%s; ping=%dms\033[0m\n", address, port, duration.Milliseconds())
		}
	}

	if continuous {
		fmt.Fprintf(output, "\033[92mPinging %s:%s continuously for %d times.\033[0m\n", address, port, count)
		for i := 0; i < count; i++ {
			ping()
			time.Sleep(1 * time.Second)
		}
		fmt.Fprintf(output, "\033[92mContinuous ping to %s:%s completed after %d times.\033[0m\n", address, port, count)
	} else {
		
		pingCount := 4
		for i := 0; i < pingCount; i++ {
			ping()
			time.Sleep(1 * time.Second)
		}
		fmt.Fprintf(output, "\033[92mPing to %s:%s completed.\033[0m\n", address, port)
	}
}

func (p *PapingCommand) Aliases() []string {
	return []string{"Paping"}
}

func (p *PapingCommand) AdminOnly() bool {
	return false
}

func init() {
	CommandMap["paping"] = &PapingCommand{}
}
