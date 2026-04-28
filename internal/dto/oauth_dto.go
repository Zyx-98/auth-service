package dto

type GoogleLoginRequest struct {
	Code  string `json:"code" validate:"required"`
	State string `json:"state" validate:"required"`
}

type OAuthCallbackResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	IsNewUser    bool   `json:"is_new_user"`
	TOTPRequired bool   `json:"totp_required"`
	TOTPToken    string `json:"totp_token,omitempty"`
	DeviceToken  string `json:"device_token,omitempty"`
}

type TOTPVerifyLoginRequest struct {
	Code        string `json:"code" validate:"required"`
	TOTPToken   string `json:"totp_token" validate:"required"`
	TrustDevice bool   `json:"trust_device,omitempty"`
}
