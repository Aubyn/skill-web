<h1 align="center">skill-web</h1>

<p align="center">
  Agent 技能管理器 — 集中管理、分组、同步本地 Agent 技能
</p>

<p align="center">
  <a href="#quickstart">快速开始</a> ·
  <a href="#features">功能</a> ·
  <a href="#usage">使用</a> ·
  <a href="#develop">开发</a> ·
  <a href="#tech">技术栈</a>
</p>

---

## 快速开始

```bash
# 下载最新 release 或自行构建
wget https://github.com/.../skill-web

# 启动（默认 127.0.0.1:7931）
./skill-web

# 打开浏览器
open http://127.0.0.1:7931
```

> 端口被占用时会自动尝试 7932 → 7933 → … → 7940，启动后第一行日志打印实际地址。

---

## 功能

### 📚 技能库

- 从本地目录递归发现含 `SKILL.md` 的技能
- 支持符号链接跟随
- 模糊搜索（后端 SQL LIKE）
- 导入时自动处理同名冲突（覆盖策略）

### 📂 技能组

- 创建/删除技能组
- **多选对话框**追加技能（搜索 + 勾选）
- 移除技能不删除技能本身
- 左侧列表计数实时更新

### 🎯 目标目录

- 管理多个目标目录（如 `~/.claude/skills/`、`~/.reasonix/skills/`）
- **展开查看**每个目录下已同步的技能列表

### 🔄 同步

- 来源：从技能库选技能 / 从技能组自动解析
- 同步前自动备份到 `~/.skill-web/backups/<ts>/`
- 采用**软链接**部署（不复制文件）
- **逐技能结果展示**：成功绿色标签、失败红色标签、已清除内容

---

## 使用流程

```
导入技能 → 建组管理 → 添加目标目录 → 执行同步 → 检查目标目录
```

![使用流程图](https://via.placeholder.com/800x200?text=skill-web+workflow)

---

## 开发

### 前置条件

- Go >= 1.22
- Node.js >= 18
- pnpm

### 本地开发

```bash
# 终端 1：启动 Go 后端
go run .

# 终端 2：启动前端 dev server（含 API 代理）
cd frontend && pnpm dev
# → http://localhost:5173 (自动代理 /api 到 7931)
```

### 构建

```bash
# 构建前端
cd frontend && pnpm build

# 构建单一二进制（内嵌前端 dist）
go build -o skill-web .
```

构建产物为 `skill-web`（Linux amd64），单一二进制无外部依赖。

### 项目结构

```
skill-web/
├── main.go                 # 入口 + 路由 + 端口回退
├── db/                     # SQLite 数据层
│   ├── db.go              # 初始化 + 全部 CRUD
│   ├── models.go          # 数据模型
│   └── schema.sql         # DDL
├── scanner/                # 技能发现
│   └── scanner.go         # 递归扫描 + 软链跟随
├── syncer/                 # 同步
│   └── syncer.go          # 备份 + 清空 + 软链部署
├── handler/                # HTTP handlers
│   ├── skills.go          # 技能库 API
│   ├── groups.go          # 技能组 API
│   ├── targets.go         # 目标目录 API + resolvePath 共享函数
│   └── sync.go            # 同步 API
└── frontend/               # React SPA
    └── src/
        ├── api/client.ts  # 类型安全 API 客户端
        ├── pages/         # 4 个页面组件
        └── components/    # 布局组件
```

---

## 技术栈

| 层 | 选择 |
|---|---|
| 后端 | **Go 1.22+** + **Echo v4** |
| 数据库 | **SQLite**（modernc.org/sqlite，零 CGO） |
| 前端 | **React 19** + **TypeScript** + **Vite** |
| UI | **Tailwind CSS 3** + 自定义暖色主题 + **Lucide** 图标 |
| 路由 | **react-router-dom v7** |
| 分发 | **单一二进制**内嵌前端 dist |

### 数据库文件

```
~/.skill-web/
├── skill-web.db          # SQLite（WAL 模式）
├── skills/               # 导入的技能副本
└── backups/              # 同步前自动备份
```

---

## 常见问题

**Q: 如何修改监听的起始端口？**  
A: 编辑 `main.go` 中 `for port := 7931;` 的起始值，重新构建。

**Q: 如何添加新目标目录？**  
A: 在 UI「目标目录」页点击「添加目录」，输入绝对路径即可。路径中的 `~` 会自动展开。

**Q: 同步后的技能不在目标目录显示？**  
A: 到「目标目录」页展开该目录查看技能列表。若提示"目录不存在"，检查路径是否正确；检查服务是否已重启（同步和查看使用了统一的路径解析）。

**Q: Windows 如何运行？**  
A: 当前版本需要交叉编译：`GOOS=windows GOARCH=amd64 go build -o skill-web.exe .`，或使用 WSL。

---

## License

MIT
