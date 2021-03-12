package config

// A Profile contains all information for connecting to a menmos cluster.
type Profile struct {
	Host     string `json:"host,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
