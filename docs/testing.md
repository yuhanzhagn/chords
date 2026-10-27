# 测试与检查

## 按变更范围选择

| 变更 | 根目录命令 | 边界 |
| --- | --- | --- |
| 文档与指南 | make check-docs | 本地内联 Markdown 链接存在性，不验证外网/锚点/语义 |
| Python / 辅助脚本语法 | make check-python | ast.parse，不 import、不创建 pyc、不启动服务 |
| backend | make test-backend | 单元与分层集成，内存 SQLite/mocks，无需真实 Redis/Kafka |
| connection | make test-connection | handler、sink、gateway、HTTP source、装配函数等；不连接真实依赖 |
| fanout | make test-fanout | 当前没有测试文件，只是编译检查 |
| 前端 | make test-frontend / make build-frontend | 单次 Jest 冒烟 / 生产构建；先 make install-frontend |
| 跨模块或 schema | make test-go，加前端 build/test | 检查四个消费者；无法代替真实消息链路 |
| 全仓基线 | make check | 不自动安装依赖；汇总以上关键检查 |
| 不写二进制的 Go 编译 | make build-go | 各模块 go build ./... |

Go 单模块命令也可直接在相应目录执行 go test ./...。精确选择包或用 `-run TestName` 定位；有新变更、失败或未解决疑点时才重复/扩大检查。

修改网关并发/连接生命周期时按需运行：

```bash
cd connection
go test -race ./cmd/server ./internal/gateway ./internal/sink
```

frontend 另有 `npm run lint` 和 `npx --no-install tsc --noEmit`。不默认运行 npm test 的 watch 模式；根 Make 已固定 CI=true、watchAll=false、runInBand。

## 测试环境说明

App 冒烟测试 mock 启动时的 token 检查 fetch，不依赖 backend。CRA 使用 Jest 27，无法按当前 protobuf 包 exports 直接解析 wire 子路径；package.json 的 Jest moduleNameMapper 指向已锁 runtime 的 CJS 文件，运行时构建仍使用正常 imports。升级 protobuf/CRA 时核对此映射。

Python 暂无 pytest 套件。新行为测试应覆盖空库、重建、重载、更新、删除与停用词边界，使用临时数据路径。ast.parse 成功不能说明依赖安装齐全或运行行为正确。

## 2026-10-27 实际验证

环境：Linux/WSL、Node 18.19.1 / npm 9.2.0、Python 3.12.3。宿主 go version 是 1.22.2，module 命令自动选择缓存中的 Go 1.24.6 / 1.24.12。

| 检查 | 结果 |
| --- | --- |
| make check / make build-go | 全部通过；汇总检查与不写二进制的构建入口均验证成功 |
| backend go test ./... | 通过，包括 service/controller 与两组分层集成 |
| connection go test ./... | 通过 |
| fanout go test ./... | 成功，无测试文件 |
| npm ci --legacy-peer-deps --no-audit --no-fund | 按锁文件安装成功，未修改依赖声明/锁文件 |
| npm run build | 通过；存在 Browserslist 数据过期提示 |
| npm run lint | 通过 |
| Jest 非交互 | 通过（1 个 App 冒烟测试）；本次修复 protobuf 子路径映射及离线 fetch mock；保留 React Router future flag 提示 |
| kubectl kustomize deploy/k8s/base | 离线渲染成功，不代表集群运行 |
| docker compose config --quiet | 未成功：本机 WSL 缺少 Docker Desktop 集成；没有启动服务 |
| 协议脚本生成到临时目录 | 成功；三份 Go 产物一致，TS 只有生成器头 2.11.2 → 2.11.5 差异；仓库产物未改 |
| Python 服务运行 | 未执行，本机缺 sklearn；未安装服务依赖 |

本次文档链接检查通过（21 个 Markdown 文件、152 条本地链接），Python/辅助脚本语法解析通过（7 个文件）；链接检查器也验证了缺失目标返回失败与忽略 fenced code 示例的行为。CI/Compose YAML 语法解析通过。完整 Compose、kind 集群、Python API、负载和端到端投递未验证；不使用本页推断运行环境已经配置完成。

## 手工端到端验收

使用 [隔离演示](development.md) 注册并登录两名用户，创建/加入房间，选择房间发送，检查发送者 pending、接收者新消息、历史分页、断线重入和其他连接存活。按 gateway → Kafka → backend → fanout → gateway → UI 查每跳证据。已知未完成项见 [现状](status.md)。

CI 当前运行三个 Go 模块测试、前端测试/构建，以及文档链接/Python 语法；Python 行为、fanout 分发语义和完整部署链路仍无自动覆盖。
