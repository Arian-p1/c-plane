package config

import (
	"time"
)

type Config struct {
	Info     *Info     `yaml:"info"`
	Logger   *Logger   `yaml:"logger"`
	NBI      *NBI      `yaml:"nbi"`
	UI       *UI       `yaml:"ui"`
	Web      *Web      `yaml:"web"`
	Database *Database `yaml:"database"`
	GenieACS *GenieACS `yaml:"genieacs"`
}

type Info struct {
	Version     string `yaml:"version,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Logger struct {
	Level           string `yaml:"level,omitempty"`
	ReportCaller    bool   `yaml:"reportCaller,omitempty"`
	File            string `yaml:"file,omitempty"`
	RotationCount   int    `yaml:"rotationCount,omitempty"`
	RotationTime    string `yaml:"rotationTime,omitempty"`
	RotationMaxAge  int    `yaml:"rotationMaxAge,omitempty"`
	RotationMaxSize int    `yaml:"rotationMaxSize,omitempty"`
}

type NBI struct {
	Scheme       string        `yaml:"scheme"`
	BindingIPv4  string        `yaml:"bindingIPv4"`
	BindingIPv6  string        `yaml:"bindingIPv6"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	TLS          *TLS          `yaml:"tls,omitempty"`
}

type UI struct {
	Scheme       string        `yaml:"scheme"`
	BindingIPv4  string        `yaml:"bindingIPv4"`
	BindingIPv6  string        `yaml:"bindingIPv6"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"readTimeout"`
	WriteTimeout time.Duration `yaml:"writeTimeout"`
	TLS          *TLS          `yaml:"tls,omitempty"`
	Theme        string        `yaml:"theme"`
}

type TLS struct {
	Cert string `yaml:"cert,omitempty"`
	Key  string `yaml:"key,omitempty"`
}

type Web struct {
	UploadDir    string `yaml:"uploadDir"`
	MaxFileSize  int64  `yaml:"maxFileSize"`
	MaxTotalSize int64  `yaml:"maxTotalSize"`
	AllowedTypes string `yaml:"allowedTypes"`
}

type Database struct {
	Type     string  `yaml:"type"`
	URL      string  `yaml:"url"`
	Name     string  `yaml:"name"`
	AuthType string  `yaml:"authType,omitempty"`
	Username string  `yaml:"username,omitempty"`
	Password string  `yaml:"password,omitempty"`
	Pool     *DBPool `yaml:"pool,omitempty"`
}

type DBPool struct {
	MaxIdleConns    int           `yaml:"maxIdleConns,omitempty"`
	MaxOpenConns    int           `yaml:"maxOpenConns,omitempty"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime,omitempty"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime,omitempty"`
}

type GenieACS struct {
	CWMPURL  string        `yaml:"cwmpUrl"`
	NBIURL   string        `yaml:"nbiUrl"`
	FSURL    string        `yaml:"fsUrl"`
	Username string        `yaml:"username,omitempty"`
	Password string        `yaml:"password,omitempty"`
	Timeout  time.Duration `yaml:"timeout"`
}
