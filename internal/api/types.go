package api

type CertRequest struct {
	VM        string `json:"vm"`
	Username  string `json:"username,omitempty"`
	PublicKey string `json:"public_key"`
}

type CertResponse struct {
	VMName      string `json:"vm_name"`
	VMSlug      string `json:"vm_slug"`
	Username    string `json:"username"`
	RouterUser  string `json:"router_user"`
	EdgeHost    string `json:"edge_host"`
	EdgePort    int    `json:"edge_port"`
	Certificate string `json:"certificate"`
	ExpiresAt   string `json:"expires_at"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"` //nolint:gosec // API field name from backend contract
}

type VerifyLoginOTPRequest struct {
	OTPSession string `json:"otp_session"`
	Code       string `json:"code"`
}

type ResendLoginOTPRequest struct {
	OTPSession string `json:"otp_session"`
}

type AuthResponse struct {
	User        User   `json:"user,omitempty"`
	Token       string `json:"token,omitempty"`
	RequiresOTP bool   `json:"requires_otp,omitempty"`
	OTPSession  string `json:"otp_session,omitempty"`
}

type User struct {
	UID      string `json:"uid"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}

type Operation struct {
	UID          string `json:"uid"`
	ResourceType string `json:"resource_type"`
	ResourceUID  string `json:"resource_uid"`
	ResourceName string `json:"resource_name"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	EventsURL    string `json:"events_url"`
}

type VMOperationResult struct {
	OperationID string `json:"operation_id"`
	VMUID       string `json:"vm_uid"`
	Name        string `json:"name"`
	Action      string `json:"action"`
	Status      string `json:"status"`
	EventsURL   string `json:"events_url"`
	IsReplay    bool   `json:"is_replay"`
	IsCompleted bool   `json:"is_completed,omitempty"`
}

type VM struct {
	UID           string `json:"uid"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	DisplayStatus string `json:"display_status"`
	PlanSlug      string `json:"plan_slug,omitempty"`
	VCPU          int    `json:"vcpu,omitempty"`
	MemoryMiB     int    `json:"memory_mib,omitempty"`
	ImageSlug     string `json:"image_slug,omitempty"`
	ImageName     string `json:"image_name,omitempty"`
	ZoneSlug      string `json:"zone_slug"`
	DiskGiB       int    `json:"disk_gib"`
}

type Workspace struct {
	UID         string `json:"uid"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}
