# WhiteCNC

A powerful and customizable Command & Control (C2) server built in Go, featuring SSH-based management, API integration, Telegram bot support, and advanced security features.

## 🚀 Features

### Core Functionality
- **SSH-Based Access**: Secure SSH server for client connections
- **MySQL Database**: Robust user management and attack logging
- **RESTful API**: HTTP API for external integrations
- **Telegram Bot**: Optional Telegram bot for remote management
- **Multi-User Support**: Role-based access control (Admin, VIP, Reseller, etc.)
- **Attack Management**: Launch and monitor network stress tests
- **Real-time Chat**: Global chat system with spam protection

### Security Features
- **Rate Limiting**: IP-based connection rate limiting
- **SSH Client Filtering**: Whitelist/blacklist specific SSH clients
- **IP Whitelisting/Blacklisting**: Control access at the network level
- **Spam Protection**: User-level spam detection and prevention
- **License Authentication**: Built-in license verification system
- **Session Management**: Secure session handling with encryption support

### Advanced Capabilities
- **Customizable Branding**: Multiple theme support with custom splash screens
- **Attack Methods**: Multiple attack vectors with configurable parameters
- **Logging System**: Comprehensive attack and user activity logging
- **Cooldown Management**: Per-user attack cooldown periods
- **Concurrent Limits**: Configurable concurrent attack limits
- **API Integration**: Full API with geo-targeting and RPS control

## 📋 Requirements

- **Go**: 1.23.4 or higher
- **MySQL**: 5.7 or higher
- **Operating System**: Linux (recommended), macOS, Windows

### Dependencies
All dependencies are managed via Go modules:
- `github.com/gliderlabs/ssh` - SSH server implementation
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram bot API
- `github.com/mattn/go-shellwords` - Shell command parsing
- `golang.org/x/crypto` - Cryptography utilities
- `golang.org/x/term` - Terminal utilities

## 🛠️ Installation

### 1. Clone the Repository
```bash
git clone https://github.com/whitethegod/WhiteCNC.git
cd WhiteCNC
```

### 2. Configure MySQL Database
Create a MySQL database and user:
```sql
CREATE DATABASE your_database_name;
CREATE USER 'your_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON your_database_name.* TO 'your_user'@'localhost';
FLUSH PRIVILEGES;
```

### 3. Configure the Application
Edit `assets/config.json` with your settings:

```json
{
  "mysql": {
    "db_host": "localhost",
    "db_name": "your_database_name",
    "db_pass": "your_password",
    "db_user": "your_user"
  },
  "cnc": {
    "port": "1337",
    "api_port": "1234",
    "connection_ip": "0.0.0.0",
    "license": "your-license-here",
    "current_theme": "neverc2"
  }
}
```

### 4. Build and Run
```bash
go mod download
go build -o whitecnc main.go
./whitecnc
```

## 📝 Configuration

### Main Configuration (`assets/config.json`)

#### Database Settings
```json
"mysql": {
  "db_host": "localhost",
  "db_name": "database",
  "db_pass": "password",
  "db_user": "user"
}
```

#### CNC Settings
```json
"cnc": {
  "port": "1337",              // SSH server port
  "api_port": "1234",          // API server port
  "connection_ip": "0.0.0.0",  // Bind address
  "license": "your-license",   // License key
  "current_theme": "neverc2",  // Theme name
  "global_cooldown": 0,        // Global cooldown in seconds
  "global_slots": 1            // Global concurrent slots
}
```

#### Security Settings
```json
"Security_Settings": {
  "Max_Connections_Per_IP": {
    "enabled": true,
    "max_connections": 3,
    "time_window_minutes": 1,
    "cooldown_minutes": 1
  },
  "Blocked_SSH_Clients": [
    "ssh2js", "ubuntu", "SSH-2.0-paramiko"
  ],
  "Whitelisted_SSH_Clients": {
    "enabled": false,
    "list": ["SSH-2.0-PuTTY", "SSH-2.0-OpenSSH"]
  }
}
```

#### Spam Protection
```json
"Spam_Protection_Settings": {
  "user_spam_protection": {
    "in_general": {
      "settings": {
        "enabled": true,
        "amount": 5,
        "per_seconds": 60
      }
    }
  }
}
```

#### Telegram Bot (Optional)
```json
"telegram": {
  "enabled": true,
  "bot_token": "YOUR_BOT_TOKEN_FROM_BOTFATHER"
}
```

#### Chat Settings
```json
"Chat_Settings": {
  "Enabled": true,
  "Max_Users": 100,
  "Max_Messages_Per_User_Per_Minute": 10,
  "Title": "=== Never SRC GLOBAL CHAT ==="
}
```

#### API Options
```json
"api_options": {
  "geo": {
    "default": "ALL",
    "list": ["CN", "JP", "US"]
  },
  "rps": {
    "default": 64,
    "min": 1,
    "max": 300
  },
  "threads": {
    "default": 5,
    "min": 1,
    "max": 100
  }
}
```

## 🎯 Usage

### Default Credentials
After first run, a default admin account is created:
- **Username**: `admin`
- **Password**: `admin`

**⚠️ Change the default password immediately!**

### SSH Connection
```bash
ssh admin@your-server-ip -p 1337
```

### Available Commands

#### General Commands
- `help` / `h` - Display help menu
- `clear` / `cls` - Clear the screen
- `exit` / `quit` - Logout from session
- `methods` / `method` - Show available attack methods
- `credits` - Show credits information
- `themes` - Change interface theme

#### Attack Commands
- `attack <target> <port> <time> <method>` - Launch an attack
- `ongoing` - View ongoing attacks
- `logs` - View attack logs
- `lookup <ip/domain>` - Lookup target information
- `ping <target>` - Ping a target
- `paping <target> <port>` - TCP ping a target

#### User Management (Admin)
- `users` - Manage users
  - `users add <username> <password>` - Create new user
  - `users remove <username>` - Delete user
  - `users list` - List all users
  - `users edit <username>` - Edit user settings
- `editall <setting> <value>` - Edit setting for all users
- `plan <username>` - View user's plan details

#### Chat Commands
- `chat` - Access global chat
- `echo <message>` - Send a message to all online users

#### Security Commands
- `password <new_password>` - Change your password
- `encrypt <text>` - Encrypt text using bcrypt

#### Configuration Commands
- `toggle <setting>` - Toggle attack settings
- `cfx` - Modify configuration settings

## 🏗️ Project Structure

```
whitecnc/
├── assets/
│   ├── blacklists/           # Blacklist configurations
│   ├── branding/             # Theme files (.tfx)
│   │   ├── milnet/
│   │   └── neverc2/
│   ├── funnel/               # Funnel configurations
│   ├── logs/                 # Log files
│   ├── public/               # Public HTML files
│   ├── config.json           # Main configuration
│   └── gradient.json         # Terminal gradient settings
├── commands/
│   └── cmds/                 # Command implementations
├── database/
│   └── Database.go           # Database operations
├── handlers/
│   ├── AttackHandler.go      # Attack request handling
│   ├── CommandHandler.go     # Command processing
│   ├── FunnelHandler.go      # Funnel management
│   └── SessionHandler.go     # SSH session handling
├── managers/
│   ├── AttackManager.go      # Attack orchestration
│   ├── FunnelManager.go      # Funnel operations
│   └── LogManager.go         # Logging operations
├── telegram/
│   └── TelegramBot.go        # Telegram bot implementation
├── utils/
│   ├── AuthenticationUtil.go # License verification
│   ├── BrandingUtil.go       # Theme rendering
│   ├── ConfigUtil.go         # Configuration loading
│   ├── SecurityUtil.go       # Security features
│   ├── SpamProtectionUtil.go # Spam detection
│   └── ...                   # Other utilities
├── main.go                   # Application entry point
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## 🔌 API Endpoints

The API server runs on the configured `api_port` (default: 1234).

### Attack API
```
GET/POST /api/attack?target=<ip>&port=<port>&time=<seconds>&method=<method>&username=<user>&key=<apikey>
```

**Parameters:**
- `target` - Target IP/domain
- `port` - Target port
- `time` - Attack duration (seconds)
- `method` - Attack method name
- `username` - API username
- `key` - API key

**Optional Parameters:**
- `geo` - Geographic region (CN, JP, US, etc.)
- `rps` - Requests per second
- `threads` - Number of threads
- `len` - Packet length

**Response:**
```json
{
  "status": "success",
  "message": "Attack sent successfully",
  "attack_id": "unique-id",
  "target": "1.2.3.4",
  "port": 80,
  "time": 60,
  "method": "HTTP-FLOOD"
}
```

## 🎨 Customization

### Creating Custom Themes

1. Create a new folder in `assets/branding/yourtheme/`
2. Add `.tfx` template files:
   - `title.tfx` - Login banner
   - `home-splash.tfx` - Main menu
   - `help.tfx` - Help screen
   - `methods.tfx` - Methods display
   - `attack-sent.tfx` - Attack confirmation
   - And more...

3. Update config:
```json
"cnc": {
  "current_theme": "yourtheme"
}
```

### Template Variables
Templates support variable substitution:
- `{{user.Username}}` - Current username
- `{{user.Expiry}}` - Account expiry
- `{{user.Admin}}` - Admin status
- `{{user.Concurrents}}` - Concurrent slots
- `{{user.Maxtime}}` - Max attack time
- And many more...

## 🔒 Security Best Practices

1. **Change Default Credentials**: Immediately change the default admin password
2. **Use Strong Passwords**: Enforce strong password policies
3. **Enable Rate Limiting**: Prevent brute force attacks
4. **Configure Firewalls**: Restrict access to trusted IPs
5. **Regular Updates**: Keep dependencies up to date
6. **Monitor Logs**: Regular log review for suspicious activity
7. **License Validation**: Ensure license authentication is active
8. **SSL/TLS**: Consider using SSL for API endpoints

## 📊 Database Schema

The application automatically creates the following tables:
- `users` - User accounts and permissions
- `attacks` - Attack history and logs
- `chat_messages` - Chat message storage
- `sessions` - Active session tracking

## 🐛 Troubleshooting

### Connection Issues
```bash
# Check if server is running
netstat -tulpn | grep 1337

# Test SSH connection
ssh -v admin@localhost -p 1337
```

### Database Issues
```bash
# Test MySQL connection
mysql -u your_user -p your_database

# Check database logs
tail -f /var/log/mysql/error.log
```

### License Issues
- Verify `license` field in `assets/config.json`
- Check `license.php` for authentication endpoint
- Review logs: `assets/logs/global_logs.log`

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## ⚠️ Legal Disclaimer

This software is provided for **educational and authorized testing purposes only**. Users are solely responsible for ensuring compliance with all applicable laws and regulations. Unauthorized access to computer systems is illegal. The developers assume no liability for misuse of this software.

## 📄 License

This project is licensed under a custom license. See the `license.php` file for details.

## 👥 Credits

Developed by **WhiteTheGod** and contributors.

## 📞 Support

For support, issues, or feature requests:
- **GitHub Issues**: [https://github.com/whitethegod/WhiteCNC/issues](https://github.com/whitethegod/WhiteCNC/issues)
- **GitHub Repository**: [https://github.com/whitethegod/WhiteCNC](https://github.com/whitethegod/WhiteCNC)

## 🔄 Version History

See commit history for detailed changes and updates.

---

**⚠️ Use Responsibly**: This tool should only be used in authorized testing environments. Always obtain proper authorization before conducting any security testing.
