package config

import (
	"sync"
	"time"
)

var (
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 1000
	// DefaultMaxConcurrent is how many commands can run at the same time in an executor pool
	DefaultMaxConcurrent = 10
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50
)

type Config struct {
	Timeout                time.Duration
	MaxConcurrentRequests  int
	SleepWindow            time.Duration
	RequestVolumeThreshold uint64
	ErrorPercentThreshold  int
}

var circuitConfig map[string]*Config
var configMutex *sync.RWMutex

func init() {
	circuitConfig = make(map[string]*Config)
	configMutex = &sync.RWMutex{}
}

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
}

// Configure applies settings for a set of circuits
func Configure(cmds map[string]CommandConfig) {
	for k, v := range cmds {
		ConfigureCommand(k, v)
	}
}

// ConfigureCommand applies settings for a circuit
func ConfigureCommand(name string, config CommandConfig) {
	configMutex.Lock()
	defer configMutex.Unlock()

	timeout := DefaultTimeout
	if config.Timeout != 0 {
		timeout = config.Timeout
	}

	max := DefaultMaxConcurrent
	if config.MaxConcurrentRequests != 0 {
		max = config.MaxConcurrentRequests
	}

	volume := DefaultVolumeThreshold
	if config.RequestVolumeThreshold != 0 {
		volume = config.RequestVolumeThreshold
	}

	sleep := DefaultSleepWindow
	if config.SleepWindow != 0 {
		sleep = config.SleepWindow
	}

	errorPercent := DefaultErrorPercentThreshold
	if config.ErrorPercentThreshold != 0 {
		errorPercent = config.ErrorPercentThreshold
	}

	circuitConfig[name] = &Config{
		Timeout:                time.Duration(timeout) * time.Millisecond,
		MaxConcurrentRequests:  max,
		RequestVolumeThreshold: uint64(volume),
		SleepWindow:            time.Duration(sleep) * time.Millisecond,
		ErrorPercentThreshold:  errorPercent,
	}
}

// GetCircuitConfig get the config of the circuit by name
func GetCircuitConfig(name string) *Config {
	configMutex.RLock()
	s, exists := circuitConfig[name]
	configMutex.RUnlock()

	if !exists {
		ConfigureCommand(name, CommandConfig{}) // use the default config
		s = GetCircuitConfig(name)
	}

	return s
}

func GetCircuitConfigMap() map[string]*Config {
	copy := make(map[string]*Config)

	configMutex.RLock()
	for key, val := range circuitConfig {
		copy[key] = val
	}
	configMutex.RUnlock()

	return copy
}
