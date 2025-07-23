package models

import (
	"time"
)

// Device represents a TR-069 CPE device
type Device struct {
	ID             string                 `json:"id" bson:"_id"`
	DeviceID       DeviceID               `json:"deviceId" bson:"_id"`
	LastInform     time.Time              `json:"lastInform" bson:"_lastInform"`
	LastBoot       time.Time              `json:"lastBoot" bson:"_lastBoot"`
	LastBootstrap  time.Time              `json:"lastBootstrap" bson:"_lastBootstrap"`
	LastRegistered time.Time              `json:"lastRegistered" bson:"_registered"`
	Root           string                 `json:"root" bson:"_root"`
	Tags           map[string]bool        `json:"tags" bson:"_tags"`
	DeviceInfo     map[string]interface{} `json:"deviceInfo" bson:"_deviceId"`
	Parameters     map[string]Parameter   `json:"parameters" bson:"-"`

	ConnectionRequest ConnectionRequest `json:"connectionRequest" bson:"connectionRequest"`
	Status            DeviceStatus      `json:"status" bson:"status"`
}

// DeviceID contains identifying information for the device
type DeviceID struct {
	Manufacturer      string `json:"manufacturer" bson:"_Manufacturer"`
	OUI               string `json:"oui" bson:"_OUI"`
	ProductClass      string `json:"productClass" bson:"_ProductClass"`
	SerialNumber      string `json:"serialNumber" bson:"_SerialNumber"`
	DeviceIDString    string `json:"deviceIdString" bson:"_DeviceId._value"`
	HardwareVersion   string `json:"hardwareVersion" bson:"_HardwareVersion"`
	SoftwareVersion   string `json:"softwareVersion" bson:"_SoftwareVersion"`
	ModelName         string `json:"modelName" bson:"_ModelName"`
	ProvisioningCode  string `json:"provisioningCode" bson:"_ProvisioningCode"`
	ManufacturerOUI   string `json:"manufacturerOUI" bson:"_ManufacturerOUI"`
	ExternalIPAddress string `json:"externalIPAddress" bson:"_ExternalIPAddress"`
	IPAddress         string `json:"ipAddress" bson:"_IPAddress"`
}

// ConnectionRequest contains connection request information
type ConnectionRequest struct {
	URL      string `json:"url" bson:"url"`
	Username string `json:"username" bson:"username"`
	Password string `json:"password" bson:"password"`
}

// DeviceStatus represents the current status of the device
type DeviceStatus struct {
	Online           bool      `json:"online" bson:"online"`
	LastSeen         time.Time `json:"lastSeen" bson:"lastSeen"`
	ConnectionStatus string    `json:"connectionStatus" bson:"connectionStatus"`
	ErrorCount       int       `json:"errorCount" bson:"errorCount"`
}

// Parameter represents a device parameter
type Parameter struct {
	Path       string                 `json:"path" bson:"path"`
	Value      interface{}            `json:"value" bson:"value"`
	Type       string                 `json:"type" bson:"type"`
	Writable   bool                   `json:"writable" bson:"writable"`
	LastUpdate time.Time              `json:"lastUpdate" bson:"lastUpdate"`
	Attributes map[string]interface{} `json:"attributes,omitempty" bson:"attributes,omitempty"`
}

// Fault represents a device fault or alarm
type Fault struct {
	ID           string `json:"id" bson:"_id"`
	DeviceID     string `json:"deviceId" bson:"deviceId"`
	DeviceSerial string `json:"deviceSerial" bson:"deviceSerial"`
	DeviceModel  string `json:"deviceModel" bson:"deviceModel"`

	Channel        string     `json:"channel" bson:"channel"`
	Code           string     `json:"code" bson:"code"`
	Message        string     `json:"message" bson:"message"`
	Detail         string     `json:"detail,omitempty" bson:"detail,omitempty"`
	Severity       string     `json:"severity" bson:"severity"`
	Timestamp      time.Time  `json:"timestamp" bson:"timestamp"`
	Expiry         *time.Time `json:"expiry,omitempty" bson:"expiry,omitempty"`
	Retries        int        `json:"retries" bson:"retries"`
	Status         string     `json:"status" bson:"status"`
	AcknowledgedBy string     `json:"acknowledgedBy,omitempty" bson:"acknowledgedBy,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty" bson:"acknowledgedAt,omitempty"`
	ResolvedBy     string     `json:"resolvedBy,omitempty" bson:"resolvedBy,omitempty"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty" bson:"resolvedAt,omitempty"`
	Tags           []string   `json:"tags,omitempty" bson:"tags,omitempty"`
}

// FaultSeverity constants
const (
	SeverityCritical = "critical"
	SeverityMajor    = "major"
	SeverityMinor    = "minor"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// FaultStatus constants
const (
	FaultStatusActive       = "active"
	FaultStatusAcknowledged = "acknowledged"
	FaultStatusResolved     = "resolved"
	FaultStatusExpired      = "expired"
)

// FaultFilter represents filtering options for faults
type FaultFilter struct {
	DeviceID  string `json:"deviceId,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Status    string `json:"status,omitempty"`
	Channel   string `json:"channel,omitempty"`
	TimeRange string `json:"timeRange,omitempty"`
}

// DeviceFilter represents filtering options for devices
type DeviceFilter struct {
	IPRange *IPRange `json:"ipRange,omitempty"`

	Manufacturer string             `json:"manufacturer,omitempty"`
	ModelName    string             `json:"modelName,omitempty"`
	ProductClass string             `json:"productClass,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	Online       *bool              `json:"online,omitempty"`
	Search       string             `json:"search,omitempty"`
	Pagination   *PaginationOptions `json:"pagination,omitempty"`
}

// IPRange represents an IP address range for filtering
type IPRange struct {
	StartIP string `json:"startIp"`
	EndIP   string `json:"endIp"`
}

// PaginationOptions represents pagination parameters
type PaginationOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	SortBy   string `json:"sortBy,omitempty"`
	SortDir  string `json:"sortDir,omitempty"`
}

// DeviceStats represents aggregated statistics for devices
type DeviceStats struct {
	TotalDevices   int `json:"totalDevices"`
	OnlineDevices  int `json:"onlineDevices"`
	OfflineDevices int `json:"offlineDevices"`

	DevicesByVendor map[string]int `json:"devicesByVendor"`
	DevicesByModel  map[string]int `json:"devicesByModel"`
	ActiveFaults    int            `json:"activeFaults"`
	CriticalFaults  int            `json:"criticalFaults"`
}

// Task represents a task to be executed on a device
type Task struct {
	ID          string                 `json:"id" bson:"_id"`
	DeviceID    string                 `json:"deviceId" bson:"device"`
	Name        string                 `json:"name" bson:"name"`
	Status      string                 `json:"status" bson:"status"`
	Fault       *Fault                 `json:"fault,omitempty" bson:"fault,omitempty"`
	Timestamp   time.Time              `json:"timestamp" bson:"timestamp"`
	QueuedAt    time.Time              `json:"queuedAt" bson:"queuedAt"`
	StartedAt   *time.Time             `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
	Retries     int                    `json:"retries" bson:"retries"`
	Provisions  []interface{}          `json:"provisions" bson:"provisions"`
	Args        map[string]interface{} `json:"args,omitempty" bson:"args,omitempty"`
}

// TaskStatus constants
const (
	TaskStatusPending   = "pending"
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)
