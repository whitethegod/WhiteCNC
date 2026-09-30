package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)


type SpamProtectionConfig struct {
	UserSpamProtection struct {
		InGeneral struct {
			Settings struct {
				Enabled    bool `json:"enabled"`
				Amount     int  `json:"amount"`
				PerSeconds int  `json:"per_seconds"`
			} `json:"settings"`
			Actions struct {
				WarnAndBlockAttack bool `json:"warn_and_block_attack"`
				GiveCaptcha        bool `json:"give_captcha"`
				LockAccount        struct {
					Message        string `json:"message"`
					DurationInDays int    `json:"duration_in_days"`
					Enabled        bool   `json:"enabled"`
				} `json:"lock_account"`
				CloseUserSession bool `json:"close_user_session"`
			} `json:"Actions"`
		} `json:"in_general"`
		PerTarget struct {
			Settings struct {
				Enabled    bool `json:"enabled"`
				Amount     int  `json:"amount"`
				PerSeconds int  `json:"per_seconds"`
			} `json:"settings"`
			Actions struct {
				WarnAndBlockAttack bool `json:"warn_and_block_attack"`
				GiveCaptcha        bool `json:"give_captcha"`
				LockAccount        struct {
					Message        string `json:"message"`
					DurationInDays int    `json:"duration_in_days"`
					Enabled        bool   `json:"enabled"`
				} `json:"lock_account"`
				CloseUserSession bool `json:"close_user_session"`
			} `json:"actions"`
		} `json:"per_target"`
	} `json:"user_spam_protection"`
}


type SpamTracker struct {
	GeneralAttacks map[string][]time.Time            
	TargetAttacks  map[string]map[string][]time.Time 
	LockedUsers    map[string]time.Time              
	mu             sync.RWMutex
}

var (
	spamConfig  *SpamProtectionConfig
	spamTracker *SpamTracker
	spamOnce    sync.Once
)


func InitSpamProtection() error {
	var err error
	spamOnce.Do(func() {
		spamConfig, err = LoadSpamProtectionConfig("assets/config.json")
		if err != nil {
			return
		}

		spamTracker = &SpamTracker{
			GeneralAttacks: make(map[string][]time.Time),
			TargetAttacks:  make(map[string]map[string][]time.Time),
			LockedUsers:    make(map[string]time.Time),
		}

		
		go cleanupOldTracking()
	})
	return err
}


func LoadSpamProtectionConfig(filePath string) (*SpamProtectionConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var fullConfig struct {
		SpamProtectionSettings SpamProtectionConfig `json:"Spam_Protection_Settings"`
	}

	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return nil, fmt.Errorf("error parsing config: %v", err)
	}

	return &fullConfig.SpamProtectionSettings, nil
}


func GetSpamProtectionConfig() *SpamProtectionConfig {
	return spamConfig
}



func CheckSpamProtection(username, target string) (bool, bool, bool, bool, string) {
	if spamConfig == nil || spamTracker == nil {
		InitSpamProtection()
	}

	spamTracker.mu.Lock()
	defer spamTracker.mu.Unlock()

	
	if unlockTime, locked := spamTracker.LockedUsers[username]; locked {
		if time.Now().Before(unlockTime) {
			remaining := time.Until(unlockTime)
			return true, true, false, false, fmt.Sprintf("Account locked due to spam. Unlock in: %s", formatDuration(remaining))
		}
		
		delete(spamTracker.LockedUsers, username)
	}

	now := time.Now()
	isSpam := false
	message := ""

	
	if spamConfig.UserSpamProtection.InGeneral.Settings.Enabled {
		if spamTracker.GeneralAttacks[username] == nil {
			spamTracker.GeneralAttacks[username] = []time.Time{}
		}

		
		cutoff := now.Add(-time.Duration(spamConfig.UserSpamProtection.InGeneral.Settings.PerSeconds) * time.Second)
		validAttacks := []time.Time{}
		for _, t := range spamTracker.GeneralAttacks[username] {
			if t.After(cutoff) {
				validAttacks = append(validAttacks, t)
			}
		}
		spamTracker.GeneralAttacks[username] = validAttacks

		
		if len(validAttacks) >= spamConfig.UserSpamProtection.InGeneral.Settings.Amount {
			isSpam = true
			message = fmt.Sprintf("Spam detected: %d attacks in %d seconds (limit: %d)",
				len(validAttacks),
				spamConfig.UserSpamProtection.InGeneral.Settings.PerSeconds,
				spamConfig.UserSpamProtection.InGeneral.Settings.Amount)
		}

		
		spamTracker.GeneralAttacks[username] = append(spamTracker.GeneralAttacks[username], now)
	}

	
	if spamConfig.UserSpamProtection.PerTarget.Settings.Enabled && target != "" {
		if spamTracker.TargetAttacks[username] == nil {
			spamTracker.TargetAttacks[username] = make(map[string][]time.Time)
		}
		if spamTracker.TargetAttacks[username][target] == nil {
			spamTracker.TargetAttacks[username][target] = []time.Time{}
		}

		
		cutoff := now.Add(-time.Duration(spamConfig.UserSpamProtection.PerTarget.Settings.PerSeconds) * time.Second)
		validAttacks := []time.Time{}
		for _, t := range spamTracker.TargetAttacks[username][target] {
			if t.After(cutoff) {
				validAttacks = append(validAttacks, t)
			}
		}
		spamTracker.TargetAttacks[username][target] = validAttacks

		
		if len(validAttacks) >= spamConfig.UserSpamProtection.PerTarget.Settings.Amount {
			isSpam = true
			message = fmt.Sprintf("Spam detected on target %s: %d attacks in %d seconds (limit: %d)",
				target,
				len(validAttacks),
				spamConfig.UserSpamProtection.PerTarget.Settings.PerSeconds,
				spamConfig.UserSpamProtection.PerTarget.Settings.Amount)
		}

		
		spamTracker.TargetAttacks[username][target] = append(spamTracker.TargetAttacks[username][target], now)
	}

	if !isSpam {
		return false, false, false, false, ""
	}

	
	shouldBlock := spamConfig.UserSpamProtection.InGeneral.Actions.WarnAndBlockAttack ||
		spamConfig.UserSpamProtection.PerTarget.Actions.WarnAndBlockAttack

	shouldLock := spamConfig.UserSpamProtection.InGeneral.Actions.LockAccount.Enabled ||
		spamConfig.UserSpamProtection.PerTarget.Actions.LockAccount.Enabled

	shouldCloseSession := spamConfig.UserSpamProtection.InGeneral.Actions.CloseUserSession ||
		spamConfig.UserSpamProtection.PerTarget.Actions.CloseUserSession

	
	if shouldLock {
		days := spamConfig.UserSpamProtection.InGeneral.Actions.LockAccount.DurationInDays
		if days == 0 {
			days = spamConfig.UserSpamProtection.PerTarget.Actions.LockAccount.DurationInDays
		}
		unlockTime := now.Add(time.Duration(days) * 24 * time.Hour)
		spamTracker.LockedUsers[username] = unlockTime

		lockMsg := spamConfig.UserSpamProtection.InGeneral.Actions.LockAccount.Message
		if lockMsg == "" {
			lockMsg = spamConfig.UserSpamProtection.PerTarget.Actions.LockAccount.Message
		}
		if lockMsg == "" {
			lockMsg = "Your account has been locked due to spam behavior"
		}
		message = fmt.Sprintf("%s (Duration: %d days)", lockMsg, days)
	}

	return true, shouldBlock, shouldLock, shouldCloseSession, message
}


func ResetUserSpamTracking(username string) {
	if spamTracker == nil {
		return
	}

	spamTracker.mu.Lock()
	defer spamTracker.mu.Unlock()

	delete(spamTracker.GeneralAttacks, username)
	delete(spamTracker.TargetAttacks, username)
	delete(spamTracker.LockedUsers, username)
}


func UnlockUser(username string) bool {
	if spamTracker == nil {
		return false
	}

	spamTracker.mu.Lock()
	defer spamTracker.mu.Unlock()

	if _, locked := spamTracker.LockedUsers[username]; locked {
		delete(spamTracker.LockedUsers, username)
		return true
	}
	return false
}


func IsUserLocked(username string) (bool, time.Time) {
	if spamTracker == nil {
		return false, time.Time{}
	}

	spamTracker.mu.RLock()
	defer spamTracker.mu.RUnlock()

	if unlockTime, locked := spamTracker.LockedUsers[username]; locked {
		if time.Now().Before(unlockTime) {
			return true, unlockTime
		}
	}
	return false, time.Time{}
}


func cleanupOldTracking() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if spamTracker == nil {
			continue
		}

		spamTracker.mu.Lock()

		now := time.Now()
		maxAge := 1 * time.Hour

		
		for username, attacks := range spamTracker.GeneralAttacks {
			validAttacks := []time.Time{}
			for _, t := range attacks {
				if now.Sub(t) < maxAge {
					validAttacks = append(validAttacks, t)
				}
			}
			if len(validAttacks) == 0 {
				delete(spamTracker.GeneralAttacks, username)
			} else {
				spamTracker.GeneralAttacks[username] = validAttacks
			}
		}

		
		for username, targets := range spamTracker.TargetAttacks {
			for target, attacks := range targets {
				validAttacks := []time.Time{}
				for _, t := range attacks {
					if now.Sub(t) < maxAge {
						validAttacks = append(validAttacks, t)
					}
				}
				if len(validAttacks) == 0 {
					delete(targets, target)
				} else {
					targets[target] = validAttacks
				}
			}
			if len(targets) == 0 {
				delete(spamTracker.TargetAttacks, username)
			}
		}

		
		for username, unlockTime := range spamTracker.LockedUsers {
			if now.After(unlockTime) {
				delete(spamTracker.LockedUsers, username)
			}
		}

		spamTracker.mu.Unlock()
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}