# 智能体实训台

智能体实训台是一个面向高校智能体课程的全栈运行治理工作区。它保留并扩展了原有 Vue 操作界面，将后端升级为 Go 服务，围绕实训班级、学员、工具版本、执行授权和审计记录组织业务流程。基础仓库不包含预设缺陷。

前端使用 Vue 3.4、TypeScript 5.7、Vite 5、Element Plus 2.4、Pinia 2 和 Vue Router 4。后端使用 Go 1.22、`net/http`、SQLite 与版本化 migration，不依赖在线服务。

## 运行

后端默认使用 SQLite，启动时自动执行版本迁移。先创建本地登录账号：

    cd backend
    $env:DATABASE_PATH="./data/campuslab.db"
    $env:BOOTSTRAP_EMAIL="operator@example.test"
    $env:BOOTSTRAP_PASSWORD="very-secure-password"
    $env:BOOTSTRAP_DISPLAY_NAME="实训运营"
    $env:BOOTSTRAP_ROLE="agent_developer"
    go run ./cmd/seed-user
    go run ./cmd/server

另开终端启动前端：

    cd frontend-admin
    npm ci
    npm run dev

浏览器地址为 http://localhost:5173，Vite 将 `/api` 显式代理到 `http://localhost:8080`。

根目录 `Dockerfile` 构建可独立运行的全栈镜像，默认入口会幂等创建演示管理员并同时启动 Go API 与 Nginx：

    docker build -t campus-agent-lab .
    docker run --rm -p 8081:80 -v campuslab_data:/data \
      -e BOOTSTRAP_EMAIL=operator@example.test \
      -e BOOTSTRAP_PASSWORD=very-secure-password \
      -e BOOTSTRAP_DISPLAY_NAME=实训运营 \
      -e BOOTSTRAP_ROLE=agent_developer campus-agent-lab

也可以执行 `docker compose up --build` 使用前后端双容器部署。两种方式的页面地址均为 http://localhost:8081。上面的本地示例账号是 `operator@example.test`，密码是 `very-secure-password`；镜像本身不包含凭据，生产部署必须通过运行时密钥注入引导密码。

## 业务流程

- 实训班级到学员：班级页维护容量、状态和指导教师，学员页完成分页筛选、入班、转班及状态更新；后端通过乐观版本和事务维护容量与唯一身份约束。
- 执行场景到运行状态：执行治理页原子创建工作区、双信任区、执行池、已验证工具版本和待授权请求，再依次完成授权、开始、完成、归档或取消；列表可按状态分页筛选。
- 风险治理：后端还提供审批任务、执行回执、策略事件及安全复核 API，为后续治理流程扩展提供持久化能力。
- 可追溯运行：服务端会话、退出撤销、角色鉴权、幂等请求、审计事件和可重试后台作业。

主要页面与 API 对应关系：

| 页面 | 公开业务路径 | API |
| --- | --- | --- |
| 运行总览 | 班级/学员摘要与最近执行 | `/training/summary`、`/trainees`、`/execution-requests` |
| 实训班级 | 班级创建、编辑、容量和删除约束 | `/cohorts` |
| 实训学员 | 入班、转班、状态更新与删除 | `/trainees`、`/cohorts/all` |
| 执行治理 | 场景原子创建与状态机推进 | `/training/execution-scenarios`、`/execution-requests/*` |
| 治理成员/个人中心 | 成员生命周期、资料和密码 | `/users`、`/auth/profile`、`/auth/password` |
| 审计记录 | 操作者、对象和请求关联查询 | `/audit` |

GET /healthz 提供存活检查，GET /readyz 检查数据库就绪状态。错误响应统一包含错误码、用户消息和请求 ID。

## 验证

    cd backend
    go test ./... -count=1
    go test -race ./... -count=1
    go vet ./...
    go build ./...
    cd ../
    go run ./scripts/measure_project.go -root . -frontend-roots frontend-admin -enforce

    cd frontend-admin
    npm ci
    npm test -- --run
    npm run typecheck
    npm run build

根目录 `Makefile` 提供同名验证入口。前端测试覆盖 API 成功与结构化错误、取消、超时、表单校验、冲突重试、路由级恢复和执行状态推进。

## 配置

后端环境变量见 `backend/.env.example`。常用项包括 `DATABASE_PATH`、`HTTP_ADDR`、`SESSION_TTL`、`APPROVAL_TTL`、`WORKER_INTERVAL`、`WORKER_BATCH_SIZE` 和 `SHUTDOWN_TIMEOUT`。前端可通过 `VITE_API_URL` 覆盖 API 基址；未设置时使用同源 `/api/v1`。

SQLite migration 位于 `backend/migrations`，同时嵌入 Go 二进制。数据库从空库按版本升级，重复启动不会重复执行；WAL、外键、忙等待和条件更新共同提供事务与并发保护。状态和后台作业都持久化，进程重启后会从数据库恢复。
