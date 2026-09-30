package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)


type APIGlobalCooldown struct {
	Enabled         bool `json:"enabled"`
	PerAttackDelayMs int `json:"per_attack_delay_ms"`
}

type APILoadingDelay struct {
	Enabled       bool `json:"enabled"`
	DelaySeconds  int  `json:"delay_seconds"`
	ShowCountdown bool `json:"show_countdown"`
}

type APILenConfig struct {
	Default int `json:"default"`
	Max     int `json:"max"`
	Min     int `json:"min"`
}

type APIThreadsConfig struct {
	Default int `json:"default"`
	Max     int `json:"max"`
	Min     int `json:"min"`
}

type APIRPSConfig struct {
	Default int `json:"default"`
	Max     int `json:"max"`
	Min     int `json:"min"`
}

type APIGeoConfig struct {
	Default string   `json:"default"`
	List    []string `json:"list"`
}

type APIOptions struct {
	APIGlobalCooldown              APIGlobalCooldown `json:"api_global_cooldown"`
	ShowHostInfoInAPIResponse      bool              `json:"show_host_information_in_api_response"`
	APILoadingDelay                APILoadingDelay   `json:"api_loading_delay"`
	Len                            APILenConfig      `json:"len"`
	Threads                        APIThreadsConfig  `json:"threads"`
	RPS                            APIRPSConfig      `json:"rps"`
	Geo                            APIGeoConfig      `json:"geo"`
}

type MiscConfig struct {
    Spinner SpinnerConfig `json:"spinner"`
}

type SpinnerConfig struct {
    Frames   []string `json:"frames"`
    SpeedMs  int      `json:"speed_ms"`
}

type TelegramConfig struct {
    Enabled  bool   `json:"enabled"`
    BotToken string `json:"bot_token"`
}

type Config struct {
    License         string `json:"-"`
    Port            string `json:"-"`
    Funnel_port     string `json:"-"`
    ConnectionIP    string `json:"-"`
    Attacks_enabled bool   `json:"-"`
    Admins_bypass   bool   `json:"-"`
    Global_cooldown int    `json:"-"`
    Global_slots    int    `json:"-"`
    DBUser          string `json:"-"`
    DBPass          string `json:"-"`
    DBHost          string `json:"-"`
    DBName          string `json:"-"`
    CurrentTheme    string `json:"-"`
    APIOptions      APIOptions `json:"-"`
    Misc            MiscConfig `json:"-"`
    Telegram        TelegramConfig `json:"-"`
    mu sync.RWMutex
}



type AuxConfig struct {
    CNC struct {
        License         string `json:"license"`
        Port            string `json:"port"`
        Funnel_port     string `json:"api_port"`
        ConnectionIP    string `json:"connection_ip"`
        Global_cooldown int    `json:"global_cooldown"`
        Global_slots    int    `json:"global_slots"`
        CurrentTheme    string `json:"current_theme"`
    } `json:"cnc"`
    MySQL struct {
        DBUser string `json:"db_user"`
        DBPass string `json:"db_pass"`
        DBHost string `json:"db_host"`
        DBName string `json:"db_name"`
    } `json:"mysql"`
    AttackSettings struct {
        AttackStatus struct {
            AttacksEnabled              bool `json:"attacks_enabled"`
            AdminsBypassDisabledAttacks bool `json:"admins_bypass_disabled_attacks"`
        } `json:"Attack_Status"`
    } `json:"Attack_Settings"`
    APIOptions APIOptions     `json:"api_options"`
    Misc       MiscConfig     `json:"misc"`
    Telegram   TelegramConfig `json:"telegram"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
    aux := AuxConfig{}
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }

    c.License = aux.CNC.License
    c.Port = aux.CNC.Port
    c.Funnel_port = aux.CNC.Funnel_port
    c.ConnectionIP = aux.CNC.ConnectionIP
    c.Attacks_enabled = aux.AttackSettings.AttackStatus.AttacksEnabled
    c.Admins_bypass = aux.AttackSettings.AttackStatus.AdminsBypassDisabledAttacks
    c.Global_cooldown = aux.CNC.Global_cooldown
    c.Global_slots = aux.CNC.Global_slots
    c.DBUser = aux.MySQL.DBUser
    c.DBPass = aux.MySQL.DBPass
    c.DBHost = aux.MySQL.DBHost
    c.DBName = aux.MySQL.DBName
    c.CurrentTheme = aux.CNC.CurrentTheme
    c.APIOptions = aux.APIOptions
    c.Misc = aux.Misc
    c.Telegram = aux.Telegram

    return nil
}


func GetConfig() *Config {
	if globalConfig == nil {
		
		InitConfig("assets/config.json")
	}
	return globalConfig
}


func (c *Config) SetCurrentTheme(themeName string) error {
	c.mu.Lock()
	c.CurrentTheme = themeName
	c.mu.Unlock()
	return c.SaveCurrentTheme("assets/config.json")
}


func (c *Config) GetCurrentTheme() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentTheme
}


func (c *Config) SaveCurrentTheme(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	if cncMap, ok := fullConfig["cnc"].(map[string]interface{}); ok {
		cncMap["current_theme"] = c.CurrentTheme
	}

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}


func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}


func (c *Config) GetConnectionIP() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ConnectionIP
}


func (c *Config) SetConnectionIP(ip string) error {
	c.mu.Lock()
	c.ConnectionIP = ip
	c.mu.Unlock()
	return c.SaveConnectionIP("assets/config.json")
}


func (c *Config) SaveConnectionIP(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	if cncMap, ok := fullConfig["cnc"].(map[string]interface{}); ok {
		cncMap["connection_ip"] = c.ConnectionIP
	}

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}


func (c *Config) ToggleAttacks() error {
	c.mu.Lock()
	c.Attacks_enabled = !c.Attacks_enabled
	c.mu.Unlock()
	return c.SaveConfig("assets/config.json")
}


func (c *Config) IsAttacksEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Attacks_enabled
}


func (c *Config) AdminsBypassDisabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Admins_bypass
}


func (c *Config) SetAdminsBypassDisabled(bypass bool) error {
	c.mu.Lock()
	c.Admins_bypass = bypass
	c.mu.Unlock()
	return c.SaveAdminsBypassConfig("assets/config.json")
}


func (c *Config) SaveAdminsBypassConfig(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	if attackSettings, ok := fullConfig["Attack_Settings"].(map[string]interface{}); ok {
		if attackStatus, ok := attackSettings["Attack_Status"].(map[string]interface{}); ok {
			attackStatus["admins_bypass_disabled_attacks"] = c.Admins_bypass
		}
	}

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}


func (c *Config) SaveConfig(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	if attackSettings, ok := fullConfig["Attack_Settings"].(map[string]interface{}); ok {
		if attackStatus, ok := attackSettings["Attack_Status"].(map[string]interface{}); ok {
			attackStatus["attacks_enabled"] = c.Attacks_enabled
		}
	}

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}


func (c *Config) GetAPIOptions() APIOptions {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.APIOptions
}


func (c *Config) SetAPIOptions(options APIOptions) error {
	c.mu.Lock()
	c.APIOptions = options
	c.mu.Unlock()
	return c.SaveAPIOptions("assets/config.json")
}


func (c *Config) SaveAPIOptions(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	
	var fullConfig map[string]interface{}
	if err := json.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	
	fullConfig["api_options"] = c.APIOptions

	
	updatedData, err := json.MarshalIndent(fullConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %v", err)
	}

	
	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	return nil
}


var globalConfig *Config

func InitConfig(filePath string) error {
	config, err := LoadConfig(filePath)
	if err != nil {
		return err
	}
	globalConfig = config
	return nil
}