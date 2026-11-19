package config

import "aibertai.local/backend/internal/envconfig"

var defaultDesignPublisherID string
var defaultDesignRespubJWTSecret string
var defaultDesignMediaJWTSecret string

type Design struct {
	DesignRespubURL       string
	DesignPublisherID     string
	DesignRespubJWTSecret string
	DesignMediaJWTSecret  string
}

func designEnv(key, fallback string) string {
	if value, ok := envconfig.LookupEnv(key); ok {
		return value
	}
	return fallback
}
func loadDesign() Design {
	return Design{
		DesignRespubURL:       designEnv("AIBERTAI_DESIGN_RESPUB_URL", ""),
		DesignPublisherID:     designEnv("AIBERTAI_DESIGN_PUBLISHER_ID", defaultDesignPublisherID),
		DesignRespubJWTSecret: designEnv("AIBERTAI_DESIGN_RESPUB_JWT_SECRET", defaultDesignRespubJWTSecret),
		DesignMediaJWTSecret:  designEnv("AIBERTAI_DESIGN_MEDIA_JWT_SECRET", defaultDesignMediaJWTSecret),
	}
}
