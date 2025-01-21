package config

type TwitterSecretData struct {
	BearerToken       string `json:"bearerToken"`
	AccessToken       string `json:"accessToken"`
	AccessTokenSecret string `json:"accessTokenSecret"`
	ConsumerKey       string `json:"consumerKey"`
	ConsumerSecret    string `json:"consumerSecret"`
}

type TrueMediaSecretData struct {
	ApiKey string `json:"apiKey"`
}

type PostgresSecretData struct {
	ConnectionString string `json:"connectionString"`
}
