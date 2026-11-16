# Magpie · 飞牛 fnOS FPK 打包说明

上游仓库：https://github.com/yetone/magpie （本工程 fork/打包自 v0.1.4289，MIT License）
用途：把 magpie 做成飞牛 fnOS 应用包（.fpk）。安装后桌面出现 **Magpie** 图标，点击**在 fnOS 桌面窗口内以 iframe 方式**打开 magpie 的 Web UI；由于走飞牛**统一网关（同源）**，**局域网和 fnConnect 远程都能打开**。

## 交付物
- `magpie-0.1.4289.fpk` —— 应用中心「手动安装」上传
- `magpie-fpk-src-0.1.4289.tar.gz` —— 打包工程源码（含官方二进制、网关代理源码/二进制、脚本、向导、图标、build.sh）

## 设计要点（按你的要求）
1. **网关只在本机 127.0.0.1**：magpie 网关默认监听 `127.0.0.1:3425`（可用向导改端口，主机固定 127.0.0.1）。本机 Agent 用 `http://127.0.0.1:3425/v1`。
2. **Web key 安装时由用户填写**：向导字段 `wizard_web_key`，要求 **≥16 位，只能是字母/数字/ `-` `.` `_` `~`**。不做随机生成；无效会阻止安装并提示。该值即 `MAGPIE_WEB_KEY`。
3. **WebUI 局域网 + fnConnect 远程都能开**：桌面入口用飞牛**统一网关**（`gatewayPrefix=/app/magpie`，`gatewaySocket=app.sock`，同源 `<当前访问域名>/app/magpie/`）。无论从局域网 IP 还是 fnConnect 域名访问飞牛桌面，iframe 都是同源，均可加载。
4. **iframe 桌面内打开**：入口 `type=iframe`（不是 `url`），不会新开浏览器标签。

## 架构
- `magpie web --addr 127.0.0.1:13430 --no-open`：页面进程；同时起网关 `127.0.0.1:<网关端口>`。
- `magpie-proxy`（Go 静态二进制，6MB）监听 `target/app.sock`，把飞牛网关转来的 `/app/magpie/...` 去掉前缀转发到 `127.0.0.1:13430`；并：
  - 注入 `magpie_web_13430=<key>` Cookie（因此 iframe URL 无需带 `?k=`，密钥不出现在地址栏）；
  - 在返回的 HTML 注入一个极小 `fetch`/`EventSource`/`WebSocket` 垫片，把前端绝对路径 `/api/...` 重写为 `/app/magpie/api/...`（magpie 前端 API 走绝对路径，子路径部署必需）；
  - 将 `Location` 重定向补回网关前缀。
- 数据持久化在 `${TRIM_PKGVAR}`；`magpie.conf`（含 web key，0600）在 `${TRIM_PKGETC}`。

## 安装
应用中心 →「手动安装」→ 选 `magpie-0.1.4289.fpk`，向导填写：
- **Web 访问密钥**：≥16 位，仅字母/数字/`-._~`（必填）
- **API 网关端口**：默认 `3425`（仅 127.0.0.1 监听）

CLI 安装（需先 `fnos-cli login`）：
```bash
fnos-cli app install-fpk /vol2/1000/DSH/magpie-0.1.4289.fpk --volume-id <卷ID> \
  --custom-parameters '[{"key":"wizard_web_key","value":"YourStrongKey_0123456789"},{"key":"wizard_gw_port","value":"3425"}]' --yes
```
装好后桌面出现 **Magpie**（仅管理员可见），点击即在桌面窗口内打开；局域网或 fnConnect 远程登录飞牛桌面后同样可打开。

## 目录结构
```
magpie/
├── manifest
├── ICON.PNG / ICON_256.PNG
├── app/
│   ├── bin/magpie            # 官方 magpie linux-amd64（nogui + web UI）
│   ├── bin/magpie-proxy      # 统一网关前缀反代（Go 静态）
│   └── ui/config             # 桌面入口（type=iframe，统一网关）
├── config/privilege, config/resource
├── cmd/                      # main/install_*/upgrade_*/uninstall_*/config_*
├── wizard/install, wizard/config
├── proxy/                    # 网关代理源码（main.go, go.mod）
└── build.sh
```

## 重新构建
```bash
tar xzf magpie-fpk-src-0.1.4289.tar.gz && cd magpie
FNPACK=/path/to/fnpack ./build.sh   # 若缺网关代理二进制且本机有 Go 会自动编译
```

## 已验证（本机）
- `fnpack build` 成功；解包后 `install_callback`（校验/写入 conf）→ `start`（magpie + 网关代理）→ `status=0` → `stop=0`
- 经 `app.sock` 模拟飞牛网关请求：`GET /app/magpie/` → 200，HTML 已注入垫片；`app.js/app.css` → 200；`/app/magpie/api/update` → 200（Cookie 注入生效）；直连 magpie 无 Cookie → 401（密钥保护有效）
- web key 校验：非法 key 安装被拒绝并提示

## 已知限制
- 受本机 fnos-cli 未登录所限，未走真实「应用中心」向导端到端安装；`${wizard_*}`/向导字段用法已按官方文档并参照本机已装的 `fn-deepseek-harness`（同为统一网关 fpk）实现。
- 仅 `x86`（linux-amd64）。ARM 需替换两个二进制并把 `manifest` 的 `platform` 改为 `arm`。
- 统一网关由飞牛按登录会话鉴权；`allUsers=false` 表示仅管理员桌面可见。


## 局域网 / Docker 容器使用网关

安装向导「网关访问范围」选择 **局域网 + Docker 容器（0.0.0.0）** 后，magpie 网关会监听 `0.0.0.0:<网关端口>`（默认 3425），本机仍可用 `127.0.0.1:3425`。因网关在网络上，**非本机来源必须带网关密钥**（否则 401）。

- 首次启动会自动生成一个名为 `fnOS-LAN` 的网关密钥，保存在 `${TRIM_PKGETC}/gateway-key.txt`（0600），也可在 **Magpie 网页 → Gateway → Gateway keys** 查看/新建。
- Docker 容器内配置（默认网桥用 `172.17.0.1`，或直接用 NAS 局域网 IP）：

```sh
# OpenAI 兼容
export OPENAI_BASE_URL=http://172.17.0.1:3425/v1     # 或 http://<NAS局域网IP>:3425/v1
export OPENAI_API_KEY=sk-magpie-key-xxxxxxxx

# Anthropic 兼容
export ANTHROPIC_BASE_URL=http://172.17.0.1:3425
export ANTHROPIC_API_KEY=sk-magpie-key-xxxxxxxx
```

> 自定义 docker 网络的“宿主机地址”不是 172.17.0.1：用容器内 `ip route | grep default` 看到的网关地址，或直接填 NAS 的局域网 IP。`host.docker.internal` 在 Linux 上需 `--add-host=host.docker.internal:host-gateway`。
- 安全：网关密钥用于鉴权，请勿把该端口暴露到公网；不需要容器/局域网访问时，可在「应用设置」里把网关访问范围改回「仅本机 127.0.0.1」。

## 关于 magpie 自带更新（`magpie update`）
- **能用**：magpie 只在 `/usr、/bin、/sbin、/opt`（dpkg/pacman/rpm/homebrew/nix）下才认为“被包管理器托管”。本包二进制位于 `/vol1/@appcenter/magpie/...`，所以它把自己当普通二进制。实测 `magpie update` 可把 **0.1.4289 原地升级到 0.1.3985**，且升级后的 0.1.3985 与本包的同源代理/垫片仍兼容（`/app/magpie/` 200、资产/API 200）。
- **fpk 场景的注意点**：
  1. 它替换的是应用目录内的二进制（属主是应用用户 `magpie`）；fnOS「应用中心」里的版本号仍显示 0.1.4289，二者会不一致；下次用应用中心安装新 fpk 会覆盖它。
  2. `magpie update` 会**重启正在运行的 magpie**（PID 改变）。本包的 `status/stop` 已用 `pgrep` 兜底，能识别并清理重启后的新进程，不会卡死。
  3. 手动在管理员 shell 里跑 `magpie update` 因应用目录属主是 `magpie` 用户通常无写权限；请从 **Magpie 网页「设置 → 更新 → 检查」** 触发（以应用用户身份运行），或交由应用中心升级。
- **本包默认**：首次启动会执行一次 `magpie update auto off`，即**关闭“后台自动下载/自动重启”**（`noAutoUpdate=true`），避免它静默替换二进制并热重启；你可以随时在 Magpie 设置里重新打开；「检查更新」和 `magpie update` 仍可手动使用。
- **推荐升级方式**：用 fnOS 应用中心安装更新版本的 fpk（我可按新版本重新打包），与系统版本管理保持一致。
