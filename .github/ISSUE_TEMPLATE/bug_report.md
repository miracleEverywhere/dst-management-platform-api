name: Bug 上报
description: Create a report to help us improve
title: "[Bug]"
labels:
- bug
  assignees: miracleEverywhere
  body:

- type: input
  id: version
  attributes:
  label: 饥荒管理平台版本
  description: 遇到问题的版本号，例如：v3.0.5
  value: 
  placeholder: v0.0.0
  validations:
  required: true

- type: input
  id: os
  attributes:
  label: 系统类型及版本
  description: 例如 Ubuntu24
  value:
  placeholder: 
  validations:
  required: true

- type: markdown
  attributes:
  value: |
  ## Describe the bug

      > 截图和日志，日志请粘贴平台运行日志

- type: textarea
  id: description
  attributes:
  label: 描述你遇到的BUG
  description: BUG描述，并描述BUG复现过程
  validations:
  required: true

- type: textarea
  id: additional-context
  attributes:
  label: 额外信息
  description: 
  placeholder: 