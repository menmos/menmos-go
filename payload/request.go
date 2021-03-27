package payload

// LoginRequest is the data sent to menmos for a login call.
type LoginRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
