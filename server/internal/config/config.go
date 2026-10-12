package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr, MongoURI, DBName, JWTSecret, DataKey, WxAppID, WxSecret string
	TplMeasure, TplMed, TplFamily, TplVisit                       string
	MissWindow                                                    time.Duration
	MakeupDays, VisitMaxDays                                      int
	VisitRemindTime                                               string
	AppEnv, CORSOrigin                                            string
}

func Env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func EnvInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return d
}

// LoadDotEnv loads KEY=VALUE pairs from path into the process env if not already set.
// Missing file is ignored. Values are not overwritten if already present.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}

func Load() Config {
	cfg := Config{
		Addr: Env("ADDR", ":8080"), MongoURI: Env("MONGO_URI", "mongodb://localhost:27017"),
		DBName: Env("MONGO_DB", "bp"), JWTSecret: Env("JWT_SECRET", ""), DataKey: Env("DATA_KEY", ""),
		WxAppID: Env("WX_APPID", ""), WxSecret: Env("WX_SECRET", ""),
		TplMeasure: Env("WX_TPL_MEASURE", ""), TplMed: Env("WX_TPL_MED", ""), TplFamily: Env("WX_TPL_FAMILY", ""), TplVisit: Env("WX_TPL_VISIT", ""),
		MissWindow: time.Duration(EnvInt("MISS_WINDOW_MINUTES", 120)) * time.Minute,
		MakeupDays: EnvInt("MAKEUP_WINDOW_DAYS", 7), VisitMaxDays: EnvInt("VISIT_MAX_DAYS", 365),
		VisitRemindTime: Env("VISIT_REMIND_TIME", "19:00"),
		AppEnv: Env("APP_ENV", ""), CORSOrigin: Env("CORS_ORIGIN", ""),
	}
	if len(cfg.JWTSecret) < 16 || len(cfg.DataKey) != 64 {
		log.Fatal("JWT_SECRET (>=16 chars) and DATA_KEY (64 hex chars = AES-256 key) are required")
	}
	if cfg.WxAppID == "" || cfg.WxSecret == "" {
		log.Fatal("WX_APPID and WX_SECRET are required")
	}
	return cfg
}
