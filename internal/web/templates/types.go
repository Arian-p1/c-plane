package templates

import (
	"time"

	"github.com/nextranet/gateway/c-plane/internal/models"
)

// BasePageData contains common data for all pages
type BasePageData struct {
	Title       string
	Theme       string
	CurrentPath string
	User        *UserInfo
}

// UserInfo contains user information for display
type UserInfo struct {
	Username string
	Role     string
	Avatar   string
}

// OverviewData contains data for the overview page
type OverviewData struct {
	BasePageData
	Stats           OverviewStats
	DevicesByVendor map[string]int
	FaultSeverity   map[string]int
	RecentFaults    []*models.Fault
	CriticalFaults  []*models.Fault
	SystemStatus    SystemStatus
}

// OverviewStats contains summary statistics for the overview
type OverviewStats struct {
	TotalDevices   int
	OnlineDevices  int
	OfflineDevices int
	ActiveFaults   int
	CriticalFaults int
	HealthScore    int
}

// ZoneOverview contains zone information for overview display

// SystemStatus contains system health information
type SystemStatus struct {
	CWMPConnected bool
	NBIConnected  bool
	FSConnected   bool
	LastCheck     time.Time
}

// DevicesPageData contains data for the devices page
type DevicesPageData struct {
	BasePageData
	Devices       []*DeviceDisplay
	TotalCount    int
	FilteredCount int
	CurrentPage   int
	PageSize      int
	TotalPages    int
	Filters       DeviceFilters
	Vendors       []string
	Models        []string
}

// DeviceDisplay contains device information for display
type DeviceDisplay struct {
	*models.Device
	StatusClass  string
	StatusText   string
	LastSeenText string
	TagList      []string
}

// DeviceFilters contains active filters for device list
type DeviceFilters struct {
	Search  string
	Vendor  string
	Model   string
	Status  string
	Tags    []string
	IPRange *models.IPRange
}

// DeviceDetailData contains data for the device detail page
type DeviceDetailData struct {
	BasePageData
	Device     *models.Device
	Parameters map[string]models.Parameter
	Tasks      []*models.Task
	Faults     []*models.Fault

	StatusHistory []StatusEvent
	IsOnline      bool
	CanManage     bool
}

// StatusEvent represents a device status change event
type StatusEvent struct {
	Timestamp time.Time
	Status    string
	Message   string
}

// FaultsPageData contains data for the faults page
type FaultsPageData struct {
	BasePageData
	Faults            []*FaultDisplay
	TotalCount        int
	ActiveCount       int
	AcknowledgedCount int
	CurrentPage       int
	PageSize          int
	TotalPages        int
	Filters           FaultFilters
	SeverityStats     map[string]int
}

// FaultDisplay contains fault information for display
type FaultDisplay struct {
	*models.Fault
	DeviceName     string
	SeverityClass  string
	StatusClass    string
	TimeAgoText    string
	CanAcknowledge bool
	CanResolve     bool
}

// FaultFilters contains active filters for fault list
type FaultFilters struct {
	DeviceID  string
	Severity  string
	Status    string
	Channel   string
	TimeRange string
}

// ZonesPageData contains data for the zones page

// ZoneDisplay contains zone information for display

// ZoneDetailData contains data for the zone detail page

// MapCoordinates represents a point on the map
type MapCoordinates struct {
	Latitude  float64
	Longitude float64
	Zoom      int
}

// MapDevice represents a device on the map

// ZoneStatistics contains detailed zone statistics

// NameValue is a generic name-value pair
type NameValue struct {
	Name  string
	Value interface{}
}

// PaginationData contains pagination information
type PaginationData struct {
	CurrentPage  int
	TotalPages   int
	PageSize     int
	TotalItems   int
	ShowFirst    bool
	ShowPrevious bool
	ShowNext     bool
	ShowLast     bool
	Pages        []int
}

// ChartData contains data for rendering charts
type ChartData struct {
	Labels   []string
	Datasets []Dataset
}

// Dataset represents a data series for charts
type Dataset struct {
	Label           string
	Data            []float64
	BackgroundColor []string
	BorderColor     []string
}

// NotificationData contains notification information
type NotificationData struct {
	Type    string // "success", "error", "warning", "info"
	Title   string
	Message string
	Actions []NotificationAction
}

// NotificationAction represents an action in a notification
type NotificationAction struct {
	Label string
	URL   string
	Class string
}

// FormData contains data for form rendering
type FormData struct {
	Action      string
	Method      string
	Fields      []FormField
	SubmitLabel string
	CancelURL   string
	Errors      map[string]string
}

// FormField represents a form field
type FormField struct {
	Name        string
	Label       string
	Type        string // "text", "select", "checkbox", etc.
	Value       interface{}
	Options     []SelectOption
	Required    bool
	Disabled    bool
	Placeholder string
	HelpText    string
	Error       string
}

// SelectOption represents an option in a select field
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// TableColumn defines a table column
type TableColumn struct {
	Key      string
	Label    string
	Sortable bool
	Width    string
	Class    string
}

// FilterPreset represents a saved filter configuration
type FilterPreset struct {
	ID      string
	Name    string
	Filters map[string]interface{}
	Default bool
}

// BreadcrumbItem represents a breadcrumb navigation item
type BreadcrumbItem struct {
	Label string
	URL   string
	Icon  string
}

// MenuItem represents a navigation menu item
type MenuItem struct {
	Label    string
	URL      string
	Icon     string
	Active   bool
	Badge    string
	Children []MenuItem
}

// ModalData contains data for modal dialogs
type ModalData struct {
	ID      string
	Title   string
	Content interface{}
	Actions []ModalAction
	Size    string // "sm", "md", "lg", "xl"
}

// ModalAction represents an action button in a modal
type ModalAction struct {
	Label   string
	Action  string
	Class   string
	Dismiss bool
}

// FilesPageData contains data for the files page
type FilesPageData struct {
	BasePageData
	Files     []FileInfo
	TotalSize int64
	Filters   FileFilters
}

// FileInfo contains file information for display
type FileInfo struct {
	ID          string
	Name        string
	Type        string
	Size        int64
	Description string
	UploadedAt  time.Time
	UploadedBy  string
	Hash        string
	MimeType    string
}

// FileFilters contains active filters for file list
type FileFilters struct {
	Type   string
	Search string
}
