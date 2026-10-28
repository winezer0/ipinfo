package config

import "testing"

func TestPoolConfigDefaultsAndLimits(t *testing.T) {
	cfg := &ServerConfig{}
	setDefaultConfig(cfg)
	if cfg.XDBPoolSize != 1 {
		t.Fatal("default pool changed")
	}
	for _, size := range []int{1, 4, 64} {
		cfg.XDBPoolSize = size
		if err := cfg.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, size := range []int{-1, 65} {
		cfg.XDBPoolSize = size
		if cfg.Validate() == nil {
			t.Fatal("invalid pool accepted")
		}
	}
}
