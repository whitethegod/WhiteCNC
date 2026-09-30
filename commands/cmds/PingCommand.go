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

type PingCommand struct{}

func (p *PingCommand) Name() string {
	return "ping"
}

func (p *PingCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) < 1 {
		fmt.Fprintln(output, "\033[91mInvalid number of arguments. Usage: ping [ip] [-t count]\033[0m")
		return
	}

	address := args[0]
	port := "80" 
	timeout := 2 * time.Second

	
	continuous := false
	count := 0
	if len(args) >= 2 && args[1] == "-t" {
		continuous = true
		if len(args) == 3 {
			var err error
			count, err = strconv.Atoi(args[2])
			if err != nil || count < 1 || count > 99 {
				fmt.Fprintln(output, "\033[91mInvalid count value. It must be a number between 1 and 99.\033[0m")
				return
			}
		} else {
			fmt.Fprintln(output, "\033[91mPlease specify the count for -t mode.\033[0m")
			return
		}
	}

	
	ping := func() {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(address, port), timeout)
		duration := time.Since(start)

		if err != nil {
			fmt.Fprintf(output, "\033[91mError pinging %s: %v\033[0m\n", address, err)
		} else {
			conn.Close()
			fmt.Fprintf(output, "\033[92m%s; ping=%dms\033[0m\n", address, duration.Milliseconds())
		}
	}

	if continuous {
		fmt.Fprintf(output, "\033[92mPinging %s continuously for %d times.\033[0m\n", address, count)
		for i := 0; i < count; i++ {
			ping()
			time.Sleep(1 * time.Second)
		}
		fmt.Fprintf(output, "\033[92mContinuous ping to %s completed after %d times.\033[0m\n", address, count)
	} else {
		
		pingCount := 4
		for i := 0; i < pingCount; i++ {
			ping()
			time.Sleep(1 * time.Second)
		}
		fmt.Fprintf(output, "\033[92mPing to %s completed.\033[0m\n", address)
	}
}

func (p *PingCommand) Aliases() []string {
	return []string{"Ping"}
}

func (p *PingCommand) AdminOnly() bool {
	return false
}

func init() {
	CommandMap["ping"] = &PingCommand{}
}
