package migrations

import _ "embed"

// RulePresets 是 000054 迁移中的预置系统规则。全新部署执行迁移时还没有管理员，
// 服务启动创建首个管理员后用它补种。
//
//go:embed 000054_rules_seed_presets.up.sql
var RulePresets string
