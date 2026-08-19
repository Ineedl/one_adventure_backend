// Package script embeds the Lua scripts used by the promotion service.
package script

import _ "embed"

// Seckill atomically validates promotion stock and per-user purchase limits.
//
//go:embed seckill.lua
var Seckill string

// SeckillCompensate restores stock and rolls back the user's purchased count
// when the corresponding Kafka event cannot be published.
//
//go:embed seckill_compensate.lua
var SeckillCompensate string
