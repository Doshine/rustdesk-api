package config

// PasskeyConfig controls the WebAuthn relying-party boundary. The RP ID and
// origins are deliberately explicit: accepting a browser supplied origin or
// deriving them from a request would make credential phishing and cross-origin
// confusion much easier.
type PasskeyConfig struct {
	Enabled                 bool     `mapstructure:"enabled"`
	RPID                    string   `mapstructure:"rp-id"`
	RPDisplayName           string   `mapstructure:"rp-display-name"`
	Origins                 []string `mapstructure:"origins"`
	RequireUserVerification bool     `mapstructure:"require-user-verification"`
}
