package config

type Config struct {
	ServiceName string        `mapstructure:"service_name"`
	Endpoint    string        `mapstructure:"endpoint"`
	Sampler     SamplerConfig `mapstructure:"sampler"`
}

type SamplerConfig struct {
	Type  string  `mapstructure:"type"`
	Ratio float64 `mapstructure:"ratio"`
}
