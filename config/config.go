package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen   string   `yaml:"listen"`
	Rules    []string `yaml:"rules"`
	MMDBPath MMDB     `yaml:"mmdb_path"`
	DNS      string   `yaml:"dns"`
}

func (c *Config) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = yaml.NewDecoder(file).Decode(c)
	if err != nil {
		return err
	}
	c.setDefault()
	return nil
}

var defaultRules = []string{
	"match,allow",
}

func (c *Config) setDefault() {
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8080"
	}
	if c.MMDBPath.Country == "" {
		c.MMDBPath.Country = "GeoLite2-Country.mmdb"
	}
	if c.MMDBPath.City == "" {
		c.MMDBPath.City = "GeoLite2-City.mmdb"
	}
	if c.MMDBPath.ASN == "" {
		c.MMDBPath.ASN = "GeoLite2-ASN.mmdb"
	}
	if c.Rules == nil {
		c.Rules = defaultRules
	}
	if c.DNS == "" {
		c.DNS = "https://1.1.1.1/dns-query"
	}
}

type MMDB struct {
	Country string `yaml:"country"`
	City    string `yaml:"city"` // 城市
	ASN     string `yaml:"asn"`  // 自治系统编号
}
