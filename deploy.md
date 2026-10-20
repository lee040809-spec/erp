# 部署指南

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
