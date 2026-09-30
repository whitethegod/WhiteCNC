package utils

import (
	"encoding/json"
	"os"
)

type Blacklist struct {
	IPs  []string `json:"ips"`
	ASNs []string `json:"asns"`
}

func ReadBlacklist(filename string) Blacklist {
	blacklist := Blacklist{
		IPs:  []string{},
		ASNs: []string{},
	}

	f, err := os.Open(filename)
	if err != nil {
		return blacklist
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	if dec.Decode(&blacklist) == nil {
		return blacklist
	}

	f.Seek(0, 0)
	var legacyIPs []string
	if dec.Decode(&legacyIPs) == nil {
		blacklist.IPs = legacyIPs
	}
	return blacklist
}

func SaveBlacklist(filename string, blk Blacklist) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(blk)
}

func AddIPToBlacklist(filename, ip string) error {
	blk := ReadBlacklist(filename)
	for _, v := range blk.IPs {
		if v == ip {
			return nil
		}
	}
	blk.IPs = append(blk.IPs, ip)
	return SaveBlacklist(filename, blk)
}

func RemoveIPFromBlacklist(filename, ip string) error {
	blk := ReadBlacklist(filename)
	tmp := []string{}
	for _, v := range blk.IPs {
		if v != ip {
			tmp = append(tmp, v)
		}
	}
	blk.IPs = tmp
	return SaveBlacklist(filename, blk)
}

func AddASNToBlacklist(filename, asn string) error {
	blk := ReadBlacklist(filename)
	for _, v := range blk.ASNs {
		if v == asn {
			return nil
		}
	}
	blk.ASNs = append(blk.ASNs, asn)
	return SaveBlacklist(filename, blk)
}

func RemoveASNFromBlacklist(filename, asn string) error {
	blk := ReadBlacklist(filename)
	tmp := []string{}
	for _, v := range blk.ASNs {
		if v != asn {
			tmp = append(tmp, v)
		}
	}
	blk.ASNs = tmp
	return SaveBlacklist(filename, blk)
}
