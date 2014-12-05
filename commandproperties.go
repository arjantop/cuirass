package cuirass

import (
	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/vaquita"
)

type CommandProperties struct {
	CircuitBreaker *circuitbreaker.CircuitBreakerProperties
}

const (
	CircuitBreakerEnabledDefault                  = true
	CircuitBreakerRequestVolumeThresholdDefault   = 20
	CircuitBreakerSleepWindowDefault              = 5000
	CircuitBreakerErrorThresholdPercentageDefault = 50
	CircuitBreakerForceOpenDefault                = false
	CircuitBreakerForceClosedDefault              = false
)

func newCommandProperties(cfg vaquita.DynamicConfig) *CommandProperties {
	pf := vaquita.NewPropertyFactory(cfg)
	propertyPrefix := pf.GetStringProperty("cuirass.config.prefix", "cuirass").Get()
	cbPrefix := ".command.default.circuitBreaker"
	return &CommandProperties{
		CircuitBreaker: &circuitbreaker.CircuitBreakerProperties{
			Enabled:                  pf.GetBoolProperty(propertyPrefix+cbPrefix+".enabled", CircuitBreakerEnabledDefault),
			RequestVolumeThreshold:   pf.GetIntProperty(propertyPrefix+cbPrefix+".requestVolumeThreshold", CircuitBreakerRequestVolumeThresholdDefault),
			SleepWindow:              pf.GetIntProperty(propertyPrefix+cbPrefix+".sleepWindowInMilliseconds", CircuitBreakerSleepWindowDefault),
			ErrorThresholdPercentage: pf.GetIntProperty(propertyPrefix+cbPrefix+".errorThresholdPercentage", CircuitBreakerErrorThresholdPercentageDefault),
			ForceOpen:                pf.GetBoolProperty(propertyPrefix+cbPrefix+".forceOpen", CircuitBreakerForceOpenDefault),
			ForceClosed:              pf.GetBoolProperty(propertyPrefix+cbPrefix+".forceClosed", CircuitBreakerForceClosedDefault),
		},
	}
}
