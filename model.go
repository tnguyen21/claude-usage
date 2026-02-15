package main

import "encoding/json"

type UsageResponse struct {
	FiveHour      *UsageBucket    `json:"five_hour"`
	SevenDay      *UsageBucket    `json:"seven_day"`
	SevenDayOpus  *UsageBucket    `json:"seven_day_opus"`
	SevenDayOAuth json.RawMessage `json:"seven_day_oauth_apps"`
	IguanaNecktie json.RawMessage `json:"iguana_necktie"`
}

type UsageBucket struct {
	Utilization float64 `json:"utilization"` // 0.0–100.0
	ResetsAt    *string `json:"resets_at"`   // ISO 8601 or null
}

type KeychainCredentials struct {
	ClaudeAiOauth *OAuthEntry `json:"claudeAiOauth"`
}

type OAuthEntry struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	SubscriptionType string `json:"subscriptionType"`
}
