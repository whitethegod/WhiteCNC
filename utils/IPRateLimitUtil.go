package utils

import (
	"sync"
	"time"
)

type IPRateLimitConfig struct {
	Enabled          bool `json:"enabled"`
	MaxConnections   int  `json:"max_connections"`
	TimeWindowMinutes int `json:"time_window_minutes"`
	CooldownMinutes  int `json:"cooldown_minutes"`
}

type IPRateLimiter struct {
	mu          sync.Mutex
	connections map[string][]time.Time
	cooldowns   map[string]time.Time
}

var (
	ipRateLimiter *IPRateLimiter
	rateLimitOnce sync.Once
)

func GetIPRateLimiter() *IPRateLimiter {
	rateLimitOnce.Do(func() {
		ipRateLimiter = &IPRateLimiter{
			connections: make(map[string][]time.Time),
			cooldowns:   make(map[string]time.Time),
		}
	})
	return ipRateLimiter
}

func (rl *IPRateLimiter) IsAllowed(ip string) bool {
	securityConfig := GetSecurityConfig()
	config := securityConfig.MaxConnectionsPerIP
	
	if !config.Enabled {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	
	if cooldownUntil, exists := rl.cooldowns[ip]; exists {
		if time.Now().Before(cooldownUntil) {
			return false
		}
		
		delete(rl.cooldowns, ip)
	}

	
	now := time.Now()
	windowStart := now.Add(-time.Duration(config.TimeWindowMinutes) * time.Minute)

	if connections, exists := rl.connections[ip]; exists {
		
		var validConnections []time.Time
		for _, connTime := range connections {
			if connTime.After(windowStart) {
				validConnections = append(validConnections, connTime)
			}
		}
		rl.connections[ip] = validConnections

		
		if len(validConnections) >= config.MaxConnections {
			
			rl.cooldowns[ip] = now.Add(time.Duration(config.CooldownMinutes) * time.Minute)
			
			delete(rl.connections, ip)
			return false
		}
	}

	
	rl.connections[ip] = append(rl.connections[ip], now)
	return true
}

func (rl *IPRateLimiter) GetRemainingCooldown(ip string) time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if cooldownUntil, exists := rl.cooldowns[ip]; exists {
		if remaining := time.Until(cooldownUntil); remaining > 0 {
			return remaining
		}
		delete(rl.cooldowns, ip)
	}
	return 0
}


func CheckIPRateLimit(ip string) bool {
	rateLimiter := GetIPRateLimiter()
	return rateLimiter.IsAllowed(ip)
}