-- 只删除未被修改过、也未被专家引用的预置规则。
DELETE FROM rules r
WHERE r.ownership_type = 'system' AND r.origin = 'local' AND r.revision = 1
  AND r.name IN ('编码行为准则', '简洁回复', '事实核查与不编造', '代码审查', '安全红线', 'Git 提交规范', '测试优先', '前端开发规范', 'Go 后端规范', 'Python 开发规范', '数据库与 SQL', 'API 设计规范', '技术文档写作', '中文排版规范', '数据分析', '工作汇报与周报', '商务邮件', 'PPT 与汇报材料')
  AND NOT EXISTS (SELECT 1 FROM expert_rules x WHERE x.rule_id = r.id);
