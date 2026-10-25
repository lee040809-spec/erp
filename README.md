# 机械加工 ERP

React + TypeScript 前端、Go + Gin 后端及 MySQL 数据库。当前为持续开发版本，不包含离线安装包。

## 目录与职责

- `platform/front/src/pages`：页面组织。
- `platform/front/src/features`：图纸、订单草稿、生产记录、经营核算、供应商分类等业务模块。
- `platform/front/src/hooks`、`services`、`utils`：共用交互、请求及计算逻辑。
- `platform/go-server/internal/handlers`：接口、权限与请求校验。
- `platform/go-server/internal/accounting`：共用月份锁、补价及送货对账。
- `platform/go-server/internal/productionflow`、`outsourcing`：生产数量流转与外协业务。
- `platform/go-server/internal/repository`、`models`：持久化及数据结构。
- `platform/go-server/cmd`：后台、备份及结构迁移工具。
- `scripts`：当前 Windows 开发环境的独立启动维护脚本。

## 开发运行

需要 Go 1.25 或以上（以 `go.mod` 为准）、Node.js 22 LTS 和可连接的 MySQL。仓库只保存源码，首次拉取需要安装依赖、编译后台并配置数据库。

前端：

```powershell
cd platform/front
npm ci
npm run dev -- --host 127.0.0.1
```

后台：将 `platform/go-server/.env.example` 复制为同目录 `.env`，填写自己的数据库信息和独立的 `JWT_SECRET`，在已初始化并完成升级的数据库上运行：

```powershell
cd platform/go-server
go run ./cmd/server
```

访问 `http://127.0.0.1:3000/platform/`，前端接口代理连接本地后台 28080 端口。后台优先读取 `.env.<GO_ENV>`（默认 `.env.local`），不存在时读取 `.env`；系统环境变量优先。

### 当前 Windows 启动脚本

现有本机已完成初始化后，可双击 `start.bat` 启动、`stop.bat` 停止。服务由 Windows 独立任务管理，关闭 Codex 不影响运行。

脚本使用 `platform/go-server/bin/erp-server-memory.exe` 和本机加密保存的数据库密码；脚本中的数据库目标属于现有开发环境，换机器部署时需要配置。源码更新后，先停止后台，再编译替换程序：

```powershell
cd platform/go-server
go build -o bin/erp-server-memory.exe ./cmd/server
```

编译文件、`.env`、`.runtime`、数据库备份和图纸原文件不进入 Git。

## 数据库升级

先备份，再检查目标库并升级。`cmd/*-migrate` 是现有测试库的受限迁移工具；未提供 `-apply` 时只检查，工具会比较升级前后的原有字段与行数据。不能将测试库命令直接用于生产库。

- 生产流转、供应商类型：`cmd/production-flow-migrate`，定义在 `internal/workflowmigration/schema.go`。
- 待补价、送货对账、费用请求去重：`cmd/accounting-workflow-migrate`，定义在 `internal/accounting/schema.sql`。
- 图纸、工序日期、下料规格、业务记录更正等升级：对应 `cmd` 工具及 `sql` 文件。

上述是既有库的增量升级，不能替代新库初始化。云服务器升级时需核对现有结构、备份并审核相应增量语句，随后自行编译发布前后端。图纸文件需与数据库一起备份。

## 验证

```powershell
cd platform/front
npm run lint
npm test
npm run build
```

```powershell
cd platform/go-server
go test ./...
go vet ./...
```

数据库集成测试默认跳过；只有明确配置测试库连接和 `RUN_FINANCE_INTEGRATION=1` 后执行。相关业务用例使用连接内临时表，不向实际业务表写入测试记录。
