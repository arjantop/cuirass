package cuirass

import (
	"time"

	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/vaquita"
)

type CommandProperties struct {
	ExecutionTimeout               vaquita.DurationProperty
	ExecutionMaxConcurrentRequests vaquita.IntProperty
	FallbackEnabled                vaquita.BoolProperty
	RequestCacheEnabled            vaquita.BoolProperty
	RequestLogEnabled              vaquita.BoolProperty
	CircuitBreaker                 *circuitbreaker.CircuitBreakerProperties
}

const (
	ExecutionTimeoutDefault               = 0
	ExecutionMaxConcurrentRequestsDefault = 100
	FallbackEnabledDefault                = true
	RequestCacheEnabledDefault            = true
	RequestLogEnabledDefault              = true

	CircuitBreakerEnabledDefault                  = true
	CircuitBreakerRequestVolumeThresholdDefault   = 20
	CircuitBreakerSleepWindowDefault              = 5000 * time.Millisecond
	CircuitBreakerErrorThresholdPercentageDefault = 50
	CircuitBreakerForceOpenDefault                = false
	CircuitBreakerForceClosedDefault              = false
)

func newCommandProperties(cfg vaquita.DynamicConfig, commandName, commandGroup string) *CommandProperties {
	pf := vaquita.NewPropertyFactory(cfg)
	propertyPrefix := pf.GetStringProperty("cuirass.config.prefix", "cuirass").Get()
	return &CommandProperties{
		ExecutionTimeout:               newDurationProperty(pf, propertyPrefix+".command", commandName, "execution.isolation.thread.timeoutInMilliseconds", ExecutionTimeoutDefault),
		ExecutionMaxConcurrentRequests: newIntProperty(pf, propertyPrefix+".command", commandGroup, "execution.isolation.semaphore.maxConcurrentRequests", ExecutionMaxConcurrentRequestsDefault),
		FallbackEnabled:                newBoolProperty(pf, propertyPrefix+".command", commandName, "fallback.enabled", FallbackEnabledDefault),
		RequestCacheEnabled:            newBoolProperty(pf, propertyPrefix+".command", commandName, "requestCache.enabled", RequestCacheEnabledDefault),
		RequestLogEnabled:              newBoolProperty(pf, propertyPrefix+".command", commandName, "requestLog.enabled", RequestLogEnabledDefault),
		CircuitBreaker: &circuitbreaker.CircuitBreakerProperties{
			Enabled:                  newBoolProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.enabled", CircuitBreakerEnabledDefault),
			RequestVolumeThreshold:   newIntProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.requestVolumeThreshold", CircuitBreakerRequestVolumeThresholdDefault),
			SleepWindow:              newDurationProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.sleepWindowInMilliseconds", CircuitBreakerSleepWindowDefault),
			ErrorThresholdPercentage: newIntProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.errorThresholdPercentage", CircuitBreakerErrorThresholdPercentageDefault),
			ForceOpen:                newBoolProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.forceOpen", CircuitBreakerForceOpenDefault),
			ForceClosed:              newBoolProperty(pf, propertyPrefix+".command", commandName, "circuitbreaker.forceClosed", CircuitBreakerForceClosedDefault),
		},
	}
}

func newBoolProperty(f *vaquita.PropertyFactory, prefix, commandName, propertyName string, defaultValue bool) vaquita.BoolProperty {
	return vaquita.NewChainedBoolProperty(f,
		prefix+"."+commandName+"."+propertyName,
		f.GetBoolProperty(prefix+".default."+propertyName, defaultValue))
}

func newIntProperty(f *vaquita.PropertyFactory, prefix, commandName, propertyName string, defaultValue int) vaquita.IntProperty {
	return vaquita.NewChainedIntProperty(f,
		prefix+"."+commandName+"."+propertyName,
		f.GetIntProperty(prefix+".default."+propertyName, defaultValue))
}

func newDurationProperty(f *vaquita.PropertyFactory, prefix, commandName, propertyName string, defaultValue time.Duration) vaquita.DurationProperty {
	return vaquita.NewChainedDurationProperty(f,
		prefix+"."+commandName+"."+propertyName,
		time.Millisecond,
		f.GetDurationProperty(prefix+".default."+propertyName, defaultValue, time.Millisecond))
}
