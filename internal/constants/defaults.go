package constants

import "time"

// Default timeout values.
const (
	DefaultTimeoutSeconds     = 30
	DefaultTimeout            = DefaultTimeoutSeconds * time.Second
	DefaultAckWaitSeconds     = 30
	DefaultAckWait            = DefaultAckWaitSeconds * time.Second
	DefaultPullTimeoutSeconds = 5
	DefaultPullTimeout        = DefaultPullTimeoutSeconds * time.Second
	DefaultStreamMaxAgeHours  = 24
	DefaultStreamMaxAge       = DefaultStreamMaxAgeHours * time.Hour
)

// Default delivery and message limits.
const (
	DefaultMaxDeliver      = 3
	DefaultPullMaxMessages = 10
	DefaultStreamReplicas  = 1
	DefaultStreamMaxMsgs   = 1000000 // 1M messages
)

// Default storage limits.
const (
	DefaultStreamMaxBytesGB = 1024 * 1024 * 1024 // 1GB
	DefaultStreamMaxBytes   = DefaultStreamMaxBytesGB
)
