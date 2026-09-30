package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Method struct {
	Enabled           bool     `json:"enabled"`
	EnabledWithFunnel bool     `json:"enabledWithFunnel"`
	Method            string   `json:"method"`
	DefaultPort       uint16   `json:"defaultPort"`
	DefaultTime       uint32   `json:"defaultTime"`
	MinTime           uint32   `json:"minTime"`
	MaxTime           uint32   `json:"maxTime"`
	Permission        []string `json:"permission"`
	Slots             int      `json:"slots"`
	API               []string `json:"api"`
}

func GetMethodsList() []Method {
	filename := "assets/funnel/funnel.json"

	cwd, err := os.Getwd()
	if err != nil {
		return []Method{}
	}

	fullPath := filepath.Join(cwd, filename)

	file, err := os.Open(fullPath)
	if err != nil {
		return []Method{}
	}
	defer func() {
		if err := file.Close(); err != nil {
		} else {
		}
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return []Method{}
	}

	var methods []Method
	err = json.Unmarshal(data, &methods)
	if err != nil {
		return []Method{}
	}

	return methods
}

func GetMethod(method string) (Method, error) {
	for _, m := range GetMethodsList() {
		if m.Method == method {
			return m, nil
		}
	}
	return Method{}, errors.New("Method not found")
}

func HasVipPermission(method string) bool {
	m, err := GetMethod(method)
	if err != nil {
		return false
	}
	for _, p := range m.Permission {
		if strings.ToLower(p) == "vip" {
			return true
		}
	}
	return false
}

func HasPrivatePermission(method string) bool {
	m, err := GetMethod(method)
	if err != nil {
		return false
	}
	for _, p := range m.Permission {
		if strings.ToLower(p) == "private" {
			return true
		}
	}
	return false
}

func HasAdminPermission(method string) bool {
	m, err := GetMethod(method)
	if err != nil {
		return false
	}
	for _, p := range m.Permission {
		if p == "ADMIN" {
			return true
		}
	}
	return false
}

func GetMethodConfig(methodName string) (*Method, error) {
	methods := GetMethodsList()
	for _, method := range methods {
		if method.Method == methodName {
			return &method, nil
		}
	}
	return nil, errors.New("Method not found")
}
