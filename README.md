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

访问 `http://127.0.0.1:3000/platform/`，前端接口代理连接本地后台 28080 端口。开发后台统一读取 `.env`；生产环境统一读取 `.env.production`。Linux 发布包使用嵌入的生产配置作为默认值，外部同名生产文件及系统环境变量可覆盖；生产环境不会回退读取开发配置。

### Linux x86-64 打包

项目发布只生成 Linux amd64 包；不再提供 Windows 发布打包。旧 Windows 本地开发启动脚本保留兼容，但数据库参数从后端 .env 读取，不再从脚本硬编码或加密 XML 读取。

在 platform/go-server/.env.production 中填写数据库、DRAWING_STORAGE_DIR、SMTP_*、FROM_EMAIL 及独立 JWT_SECRET，设置 GO_ENV=production。邮件未配置时 SMTP_HOST 留空。配置文件不进入 Git。先验证并构建前端，再在 Go 后端目录执行：

```sh
cd platform/front
npm ci
npm run lint
npm test
npm run build
cd ../go-server
go run ./cmd/package-linux -arch amd64
```

输出位于根目录 releases/erp-linux-amd64-时间.tar.gz，包含嵌入生产配置的后端、前端静态文件、systemd 和 Nginx 示例。包内不附带独立 .env 文件。私有配置临时资源及编译缓存构建后清理，发布包和实际配置均已加入 Git 排除规则。

嵌入配置不是加密；拿到二进制的人可能提取数据库及邮箱密码，因此发布包只能私下交付，不能上传公开仓库。修改嵌入配置需重新打包；也可在运行目录放置 .env.production 覆盖后重启。旧图纸、数据库及数据迁移不由此包自动复制或执行。安装步骤见包内 INSTALL.txt 和 deploy.md。

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
