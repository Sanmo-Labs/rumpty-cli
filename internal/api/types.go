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
