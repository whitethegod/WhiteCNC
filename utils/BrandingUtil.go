package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gliderlabs/ssh"
)


var globalGradients map[string]map[string]interface{}


func Init() {
	gradientFile := "./assets/gradient.json"
	file, err := os.ReadFile(gradientFile)
	if err != nil {
		fmt.Printf("Error loading gradient file: %v\n", err)
		globalGradients = make(map[string]map[string]interface{})
		return
	}

	err = json.Unmarshal(file, &globalGradients)
	if err != nil {
		fmt.Printf("Error parsing gradient file: %v\n", err)
		globalGradients = make(map[string]map[string]interface{})
	}
}


func hexToRGB(hex string) (int, int, int, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %s", hex)
	}

	r, err := strconv.ParseInt(hex[1:3], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	g, err := strconv.ParseInt(hex[3:5], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	b, err := strconv.ParseInt(hex[5:7], 16, 0)
	if err != nil {
		return 0, 0, 0, err
	}

	return int(r), int(g), int(b), nil
}

func applyGradient(text, gradientName string) string {
	gradient, exists := globalGradients[gradientName]
	if !exists {
		return text
	}

	fromColor, fromExists := gradient["from_color"].(string)
	toColor, toExists := gradient["to_color"].(string)
	
	
	background := false
	if bg, ok := gradient["background"].(bool); ok {
		background = bg
	}
	
	if !fromExists || !toExists {
		return text
	}

	
	r1, g1, b1, err := hexToRGB(fromColor)
	if err != nil {
		return text
	}

	r2, g2, b2, err := hexToRGB(toColor)
	if err != nil {
		return text
	}

	
	var result strings.Builder
	runes := []rune(text)
	length := len(runes)
	
	
	if length == 0 {
		return text
	}

	for i, char := range runes {
		var t float64
		if length == 1 {
			t = 0.0
		} else {
			t = float64(i) / float64(length-1)
		}
		
		r := int(float64(r1) + t*float64(r2-r1))
		g := int(float64(g1) + t*float64(g2-g1))
		b := int(float64(b1) + t*float64(b2-b1))

		if background {
			result.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm%c", r, g, b, char))
		} else {
			result.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm%c", r, g, b, char))
		}
	}

	
	result.WriteString("\x1b[0m")

	return result.String()
}


func GetCurrentTheme(session ssh.Session) string {
	config := GetConfig()
	if config != nil && config.GetCurrentTheme() != "" {
		return config.GetCurrentTheme()
	}
	return "default"
}


func GetBrandingFile(session ssh.Session, filename string) string {
	currentTheme := GetCurrentTheme(session)
	
	
	themePath := fmt.Sprintf("./assets/branding/%s/%s", currentTheme, filename)
	return themePath
}



func Branding(session ssh.Session, filename string, content map[string]interface{}) string {
    
    name := GetBrandingFile(session, filename + ".tfx")
    
    file, err := os.ReadFile(name)
    if err != nil {
        log.Printf("[BRANDING ERROR] Failed to read file %s: %v", name, err)
        return ""
    }
    fileContent := string(file)

    
    re := regexp.MustCompile(`<<\$?(\w+)\((\d+)\)>>`)
    fileContent = re.ReplaceAllStringFunc(fileContent, func(match string) string {
        submatches := re.FindStringSubmatch(match)
        if len(submatches) < 3 {
            return match
        }

        funcName := submatches[1]
        argStr := submatches[2]

        arg, err := strconv.Atoi(argStr)
        if err != nil {
            return match
        }

        
        if funcName == "sleep" {
            return fmt.Sprintf("<<SLEEP(%d)>>", arg)
        }

        return match
    })

    
    replacedCount := 0
    for key, value := range content {
        placeholder := "<<$" + key + ">>"
        
        
        var strValue string
        switch v := value.(type) {
        case string:
            strValue = v
        case int:
            strValue = strconv.Itoa(v)
        case int64:
            strValue = strconv.FormatInt(v, 10)
        case float64:
            strValue = strconv.FormatFloat(v, 'f', -1, 64)
        case bool:
            strValue = strconv.FormatBool(v)
        case func(int):
            
            continue
        default:
            
            continue
        }

        
        if strings.Contains(fileContent, placeholder) {
            fileContent = strings.ReplaceAll(fileContent, placeholder, strValue)
            replacedCount++
        }
    }

    
    gradientRegex := regexp.MustCompile(`<gradient name="([^"]+)">(.+?)</gradient>`)
    fileContent = gradientRegex.ReplaceAllStringFunc(fileContent, func(match string) string {
        submatches := gradientRegex.FindStringSubmatch(match)
        if len(submatches) < 3 {
            return match
        }

        gradientName := submatches[1]
        text := submatches[2]
        return applyGradient(text, gradientName)
    })

    var result strings.Builder
    lastIndex := 0
    sleepRegex := regexp.MustCompile(`<<SLEEP\((\d+)\)>>`)

    
    for _, match := range sleepRegex.FindAllStringSubmatchIndex(fileContent, -1) {
        result.WriteString(fileContent[lastIndex:match[0]])

        SendMessage(session, result.String(), false)

        durationStr := fileContent[match[2]:match[3]]
        duration, err := strconv.Atoi(durationStr)
        if err != nil {
            SendMessage(session, fileContent[match[0]:match[1]], false)
            continue
        }

        time.Sleep(time.Duration(duration) * time.Millisecond)

        result.Reset()
        lastIndex = match[1]
    }

    result.WriteString(fileContent[lastIndex:])

    return result.String()
}