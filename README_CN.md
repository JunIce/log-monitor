# 日志分析系统

基于 Go 的日志查询 Web 服务器，提供 Bash 提取脚本用于高效的日志分析和可视化。

## 概述

该系统提供 Web 界面用于查询和分析日志文件，具备以下功能：

- 基于项目的日志管理
- 时间范围过滤查询
- 关键词搜索
- 深色主题 UI
- RESTful API 日志操作

## 项目结构

```
log-analysis/
├── public/           # 前端静态文件
│   ├── index.html    # 主日志分析界面
│   ├── settings.html # 项目配置管理器
│   └── style.css     # 共享样式
├── logs/             # 日志存储目录（自动创建）
│   └── 2026-04-XX/   # 日期子目录
├── main.go           # Go 后端服务器
├── extract_log_time_range.sh # Bash 提取脚本
└── AGENTS.md        # 运维指南
```

## 安装配置

### 环境要求
- Go 1.21+
- Bash（用于时间范围提取）
- Git（可选，用于版本控制）

### 快速开始

1. **运行开发服务器**
```bash
go run . -port 8888
```

2. **构建生产版本**
```bash
go build -o log-server.exe . && ./log-server.exe -port 8888
```

3. **重新生成时间索引**（需要 bash/git bash）
```bash
bash extract_log_time_range.sh
```

## API 接口

### 日志类型和日期
```http
GET /api/log_types
```
**响应：**
```json
{
  "log_types": ["sys-info", "error", "access"],
  "dates": ["2026-04-19", "2026-04-20", "2026-04-21"]
}
```

### 日志查询
```http
GET /api/query?log_type=sys-info&start_time=09:00:00.000&end_time=12:00:00.000&date=2026-04-19
```

### 日志内容
```http
GET /api/log_content?filename=...&date=...&start_time=...&end_time=...&keyword=error
```

### 项目 API
```http
GET /api/projects     # 获取所有项目
POST /api/projects   # 添加新项目
PUT /api/projects    # 更新项目
DELETE /api/projects # 删除项目
```

## 项目管理

### 设置界面

访问项目管理器：`http://localhost:8888/settings.html`

**项目配置：**
- **项目名称**：日志集合的标识名称
- **日志目录**：日志存储路径（如 `logs/prod`）
- **索引文件**：时间范围索引文件（默认为 `time_ranges.json`）

### 操作
- 添加新项目及自定义日志目录
- 编辑现有项目配置
- 删除项目（永久移除）
- 项目设置持久化存储

## 日志格式要求

### 文件结构
日志必须存储在：`logs/{YYYY-MM-DD}/{log_type}.{date}.{seq}.log`

### 时间戳格式
每行日志首 token 必须为：`HH:mm:ss.SSS`

### 提取规则
- 脚本使用 `grep "^[0-9]"` 过滤（跳过 Java 堆栈跟踪）
- 时间范围匹配使用重叠逻辑：`entryFirst <= queryEnd && entryLast >= queryStart`

## 前端功能

### 主界面（index.html）
- 文件列表导航
- 日志内容查看器
- 时间范围过滤控制
- 实时关键词搜索
- 全屏最大化功能
- 深色主题支持

### 设置界面（settings.html）
- 项目管理仪表板
- 表单输入验证
- 消息通知
- 响应式网格布局
- 编辑/删除确认对话框

## 端口配置

- 默认：8888（避免 8080/8089/9999，因 Windows 冲突）
- 自定义端口：`go run . -port 4567`

## 开发

### 代码结构
- 后端：Go 服务器实现 REST API
- 前端：HTML/CSS/JS 深色主题支持
- 脚本：Bash 日志时间范围提取工具

### 相关命令

```bash
# 运行服务器（开发模式）
go run . -port 8888

# 构建（生产模式）
go build -o log-server.exe . && ./log-server.exe -port 8888

# 重新生成���间索引（需要 bash/git bash）
bash extract_log_time_range.sh
```

## Language

- [English](README.md)
- [中文](README_CN.md)

## License

[MIT 许可证](LICENSE) - 查看 LICENSE 文件了解详情