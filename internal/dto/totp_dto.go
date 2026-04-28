package dto

type TOTPSetupRequest struct {
}

type TOTPSetupResponse struct {
	Secret   string `json:"secret"`
	QRCode   string `json:"qr_code"`
	OTPAuth  string `json:"otp_auth"`
	Verified bool   `json:"verified"`
}

type TOTPVerifyRequest struct {
	Code        string `json:"code" validate:"required,len=6"`
	TrustDevice bool   `json:"trust_device,omitempty"`
}

type TOTPVerifyResponse struct {
	Verified bool `json:"verified"`
}

type TOTPDisableRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

type TOTPDisableResponse struct {
	Disabled bool `json:"disabled"`
}
