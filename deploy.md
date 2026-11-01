# 部署指南

## 图纸文件目录配置（2026-11-01）

上传的图纸原文件保存在 Go 后端所在服务器的磁盘中；数据库保存文件标识、名称、版本及零件关联。预览、打印和下载仍通过带权限校验的 API 读取，不要将图纸目录配置为 Nginx 公开静态目录。

在服务器后端配置文件 `/opt/erp/.env` 中增加下面一行（如已使用 `/opt/erp/.env.production`，则在该文件中设置）：

```dotenv
DRAWING_STORAGE_DIR=/var/lib/machining-erp/drawings
```

该目录应独立于程序发布目录，升级时保留。由实际运行 ERP 的 Linux 用户创建并赋予读写权限；例如运行用户为 `erp` 时，由管理员执行：

```bash
sudo install -d -m 700 -o erp -g erp /var/lib/machining-erp/drawings
```

如实际运行用户不同，请替换 `erp`。本地 Windows 配置文件为 `platform/go-server/.env`，当前路径为 `F:/code/machining-erp-main/platform/go-server/data/drawings`，兼容已有文件；可改成其他专用数据目录，路径有空格时用双引号包裹，建议使用正斜杠。

配置读取规则：环境变量优先；其次读取当前工作目录的 `.env.<GO_ENV>`（未设置 GO_ENV 时为 `.env.local`）；该文件不存在时读取 `.env`。两份配置文件不会合并，请在实际选用的文件中设置目录。未配置时兼容原目录 `data/drawings`；相对路径以 Go 服务的工作目录为基准，正式服务器推荐绝对路径。修改配置后需重启后端，日志会打印最终绝对目录；路径不可用或不可写时明确报错，不会自动换到其他目录。

**已有图纸换目录：** 先停止上传并停止后端，备份数据库及原图纸目录，将原目录完整复制到新目录，保留两位前缀子目录及文件标识，核对文件数量、大小和哈希、设置运行用户权限，再修改配置、启动服务并抽查旧图预览及新上传。旧目录在验证完成前保留。仅修改配置不会自动搬迁文件，也不需要修改数据库表结构或 storage_key。恢复旧目录配置即可回退（若切换后已有新上传，需先将新增文件同步回旧目录）。

本地与云端是不同文件系统，即使连接同一数据库也不会自动共享图纸。需在维护窗口统一复制文件并保持备份，或让各后端使用同一可靠共享存储；不要让两个不同文件目录上的后端同时创建关联图纸记录。本次修改不自动同步或移动云端文件。

备份时应同时备份数据库与该目录。历史内嵌图纸迁移工具支持环境变量 `DRAWING_STORAGE_DIR`，也可显式传 `-storage /var/lib/machining-erp/drawings`；命令行工具不自动加载 `.env`，执行时需自行导出环境变量或指定参数。

## 构建 (本地)

```bash
# 前端
Push-Location platform/front; npm run build; Pop-Location

# 后端 (Windows 交叉编译 Linux)
Push-Location platform/go-server; $env:CGO_ENABLED="0"; $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/server_linux ./cmd/server/main.go; Pop-Location


$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/server_linux ./cmd/server

```

---

## 上传 (本地)

```bash
scp platform/go-server/bin/server_linux root@8.145.45.155:/opt/erp/server
scp platform/go-server/.env.production root@8.145.45.155:/opt/erp/.env
scp -r platform/front/dist/* root@8.145.45.155:/var/www/erp/
```

---

## 重启服务 (服务器)

```bash
# 登录服务器
ssh root@8.145.45.155

# 删除旧进程
pkill -f './server'

# 赋予权限
chmod +x /opt/erp/server

# 启动服务
cd /opt/erp && GO_ENV=production nohup ./server > server.log 2>&1 &
```

---

## 查看状态 (服务器)

```bash
ps aux | grep server
tail -50 /opt/erp/server.log
```

## 重启 Nginx：

```bash
sudo nginx -t && sudo nginx -s reload
```


## nginx位置 /etc/nginx
### 合格与交货登记修改（2026-10-17）

升级此功能前先备份数据库，然后执行 `sql/019_production_record_changes.sql`（Go 服务目录内也提供同名 SQL）。只新增 `production_record_changes` 修改历史表，原表及原数据保持不变；也可用现有 production-flow-migrate 增量迁移工具，执行后会核对原列与原数据哈希。

在“办理零件 → 2 确认合格 / 3 登记交货”中，展开“查看 / 修改已登记记录”，选择记录修改数量、日期，修改原因始终必填。原记录编号、原始单价快照和费用关联保持不变；合格数量变更会重算订单数量以内的计产值数量及备用数量，日期变更更新相应月份的统计和关联费用归属。已结账月份无法直接改动。

修改记录与关联汇总在同一事务中保存。涉及已交货数量、批次投产上限、末道已完工数量或时间顺序冲突时，整次修改回退，不会只改一半。旧日期未知的交货累计记录可在核实原单据后补正日期，但不会自动拆成多次交货。

### 日常启动检查（2026-10-20）

日常启动默认只读取基础表，不执行自动建表或改表。网络读取超时不会被当成“表不存在”；连接和读取检查遇到短暂网络错误时最多尝试3次，账号、权限或缺表错误直接返回。首次初始化空库时，管理员可临时设置 `ERP_AUTO_MIGRATE=true` 创建核心表；业务表仍须按对应 SQL 迁移。不要在日常启动环境中保留该初始化开关。

双击 `start.bat` 由 Windows 独立运行 ERP。启动失败窗口会显示最近的后端错误和日志位置；启动器会等待后端有限重试完成。编译更新后使用 `stop.bat` 停止现有服务，再替换后端程序并重新启动。
