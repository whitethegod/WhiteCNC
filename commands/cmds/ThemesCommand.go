package cmds

import (
	"arismcnc/database"
	"arismcnc/utils"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gliderlabs/ssh"
)

type ThemesCommand struct{}

func (c *ThemesCommand) Name() string {
	return "themes"
}

func (c *ThemesCommand) Execute(session ssh.Session, db *database.Database, args []string, output io.Writer) {
	if len(args) == 0 {
		c.showMainMenu(output)
		return
	}

	subCommand := strings.ToLower(args[0])
	
	switch subCommand {
	case "list":
		c.listThemes(output)
	case "change", "set":
		c.changeTheme(session, output)
	case "example":
		c.showExample(session, output)
	default:
		c.showMainMenu(output)
	}
}

func (c *ThemesCommand) showMainMenu(output io.Writer) {
	fmt.Fprintln(output, "+--------------------------------------------------+")
  fmt.Fprintln(output, "| themes list                                      |")
  fmt.Fprintln(output, "| themes change                                    |")
  fmt.Fprintln(output, "| themes example                                   |")
	fmt.Fprintln(output, "+--------------------------------------------------+")
	fmt.Fprintf(output, "# Your current theme: %-29s #\n", c.getCurrentTheme())
	fmt.Fprintln(output, "+--------------------------------------------------+")
}

func (c *ThemesCommand) listThemes(output io.Writer) {
	themes := c.getAvailableThemes()
	
	fmt.Fprintln(output, "  #   Name      Description")
	fmt.Fprintln(output, " --- --------- ------------------------")
	
	for i, theme := range themes {
		fmt.Fprintf(output, "  %d   %-9s %s\n", i+1, theme.Name, theme.Description)
	}
	
	fmt.Fprintf(output, "\nYour current theme: %s\n", c.getCurrentTheme())
	fmt.Fprintln(output, "To change your current theme: themes set")
}

func (c *ThemesCommand) changeTheme(session ssh.Session, output io.Writer) {
	fmt.Fprint(output, "New Theme Name (leave empty to cancel): ")
	
	themeName, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading input.")
		return
	}
	
	themeName = strings.TrimSpace(themeName)
	if themeName == "" {
		fmt.Fprintln(output, "Action was cancelled.")
		return
	}
	
	
	if c.isValidTheme(themeName) {
		err := c.setCurrentTheme(themeName)
		if err != nil {
			fmt.Fprintf(output, "Error changing theme: %s\n", err)
			return
		}
		fmt.Fprintf(output, "Theme changed to: %s\n", themeName)
		fmt.Fprintln(output, "\033[33mPlease restart the CNC for changes to take effect.\033[0m")
	} else {
		fmt.Fprintf(output, "Theme '%s' not found. Use 'themes list' to see available themes.\n", themeName)
	}
}

func (c *ThemesCommand) showExample(session ssh.Session, output io.Writer) {
	fmt.Fprint(output, "Theme Name (leave empty to cancel): ")
	
	themeName, err := utils.ReadLine(session)
	if err != nil {
		fmt.Fprintln(output, "Error reading input.")
		return
	}
	
	themeName = strings.TrimSpace(themeName)
	if themeName == "" {
		fmt.Fprintln(output, "Action was cancelled.")
		return
	}
	
	
	if c.isValidTheme(themeName) {
		themePath := c.getThemePath(themeName)
		files, _ := c.getThemeFiles(themePath)
		
		fmt.Fprintf(output, "Theme: %s\n", themeName)
		fmt.Fprintf(output, "Path: %s\n", themePath)
		fmt.Fprintf(output, "Available files in this theme:\n")
		
		for _, file := range files {
			fmt.Fprintf(output, "  - %s\n", file)
		}
	} else {
		fmt.Fprintf(output, "Theme '%s' not found. Use 'themes list' to see available themes.\n", themeName)
	}
}

func (c *ThemesCommand) getCurrentTheme() string {
	config := utils.GetConfig()
	if config != nil {
		return config.GetCurrentTheme()
	}
	return "default"
}

func (c *ThemesCommand) getAvailableThemes() []Theme {
	themes := []Theme{}
	brandingPath := "./assets/branding"
	
	
	entries, err := os.ReadDir(brandingPath)
	if err != nil {
		
		return []Theme{
			{Name: "default", Description: "The default SSN theme."},
		}
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			themeName := entry.Name()
			description := c.getThemeDescription(themeName)
			themes = append(themes, Theme{
				Name:        themeName,
				Description: description,
			})
		}
	}
	
	
	if len(themes) == 0 {
		themes = append(themes, Theme{
			Name:        "default",
			Description: "The default SSN theme.",
		})
	}
	
	return themes
}

func (c *ThemesCommand) getThemeDescription(themeName string) string {
	
	descriptions := map[string]string{
		"default": "The default SSN theme.",
		"neverc2": "NeverC2 custom theme.",
		"dark":    "Dark theme with black background.",
		"blue":    "Blue colored theme.",
	}
	
	if desc, exists := descriptions[themeName]; exists {
		return desc
	}
	
	return "Custom theme."
}

func (c *ThemesCommand) isValidTheme(themeName string) bool {
	themePath := c.getThemePath(themeName)
	_, err := os.Stat(themePath)
	return err == nil
}

func (c *ThemesCommand) getThemePath(themeName string) string {
	return fmt.Sprintf("./assets/branding/%s", themeName)
}

func (c *ThemesCommand) getThemeFiles(themePath string) ([]string, error) {
	files := []string{}
	
	entries, err := os.ReadDir(themePath)
	if err != nil {
		return files, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tfx") {
			files = append(files, entry.Name())
		}
	}
	
	return files, nil
}

func (c *ThemesCommand) setCurrentTheme(themeName string) error {
	config := utils.GetConfig()
	if config != nil {
		return config.SetCurrentTheme(themeName)
	}
	return fmt.Errorf("config not initialized")
}

func (c *ThemesCommand) AdminOnly() bool {
	return false
}

func (c *ThemesCommand) Aliases() []string {
	return []string{"theme", "themes"}
}


type Theme struct {
	Name        string
	Description string
}

func init() {
	CommandMap["themes"] = &ThemesCommand{}
}