# qqcliput

QQ 聊天窗口 OCR 监控守护程序。当 QQ 为最前窗口时，自动捕获窗口内容并运行 Vision OCR，提取结构化消息输出到控制台。

## 设计目标

- **不保存文件** — 纯内存处理，不写入磁盘
- **守护进程** — 永久运行，QQ 窗口关闭后自动重试
- **QQ 主聊天窗口** — 自动识别并跟踪最新聊天窗口
- **结构化消息** — 提取群组、发送者、时间、内容，以 JSON 格式输出
- **增量输出** — 仅输出新增消息（基于坐标聚类去重）
- **菜单栏图标** — 系统托盘显示状态："运行中" / "等待 QQ 窗口..."
- **macOS 10.15+** — 使用 CoreGraphics + Vision，无外部依赖

## 架构

```
qqcliput (Go binary)
  │
  ├── find_qq_window()    ── 阻塞重试，每 5s 枚举窗口，取 QQ 主聊天窗口
  ├── is_qq_frontmost()   ── CGWindowList 判断最前窗口是否为 QQ
  ├── ocr_window(wid)     ── 截图 → Vision OCR → JSON (含坐标)
  │
  └── capture loop (1s ticker):
        ├── OCR text → dedup → 输出 JSON
        ├── 更新菜单栏状态
        └── 维护 5 分钟环形缓冲区
```

## 构建

```bash
./build.sh
# 输出: qqcliput.app
```

## 运行

```bash
./qqcliput.app/Contents/MacOS/qqcliput
```

## 权限

首次运行 macOS 自动弹出权限请求：

1. **屏幕录制** — System Settings → Privacy → Screen Recording

*(无需辅助功能权限)*

## 文件结构

| 文件 | 语言 | 用途 |
|---|---|---|
| `go.mod` | Go | 模块定义 + systray 依赖 |
| `main.go` | Go | 菜单栏图标、信号处理、入口 |
| `capture.go` | Go | 捕获循环：前台检测 → OCR → 去重 → 输出 |
| `cgo.go` | Go + CGo | CGo 桥接 ObjC 动态库 |
| `ringbuffer.go` | Go | 5 分钟环形缓冲区 |
| `icon.go` | Go | 菜单栏图标生成 |
| `qqcliput.h` | C | C ABI 声明 |
| `qqcliput.m` | ObjC | CoreGraphics 截图 + Vision OCR |
| `Info.plist` | XML | App Bundle 元数据 |
| `build.sh` | Shell | 编译脚本 |

## 当前实现状态

- [x] 菜单栏图标 + 状态显示
- [x] QQ 前后台自动检测
- [x] OCR 中文 + 英文识别
- [x] 去重，仅输出新文本
- [x] 5 分钟环形缓冲区
- [x] 窗口关闭后自动重试

## 待实现

### 结构化消息提取

- [ ] **OCR 输出带坐标** — `ocr_window` 返回 JSON 数组，每项含 `text` + 归一化坐标 `x/y/w/h`
- [ ] **Y 坐标聚类** — 垂直间距 < 阈值的文字块合并为同一条消息
- [ ] **字段推断** — 基于文字长度、位置、模式分割群组名、发送者、时间、内容
- [ ] **消息类型识别** — 推断每条消息的类型：

| 消息类型 | 识别依据 |
|---|---|
| `text` | 正常文字内容 |
| `image` | 内容为空或仅含 `[图片]`/文件名 |
| `video` | 包含视频时长格式 `00:00` 或 `[视频]` |
| `file` | 包含 `[文件]` 文件名 + 大小后缀 |
| `sticker` | 包含 `[表情]`/`[Sticker]`/`[动画]` |
| `system` | 系统提示（如 `"你撤回了一条消息"`） |

- [ ] **增量检测** — 内容哈希去重，每条消息仅输出一次，不重复历史消息
- [ ] **JSON 输出** — 符合以下格式：

```json
{"group":"工作群","sender":"张三","time":"10:30","type":"text","content":"好的"}
{"group":"工作群","sender":"李四","time":"10:32","type":"image"}
{"group":"工作群","sender":"王五","time":"10:33","type":"video","duration":"00:15"}
{"group":"工作群","sender":"赵六","time":"10:34","type":"sticker"}
{"group":"工作群","sender":"","time":"10:35","type":"system","content":"你撤回了一条消息"}
```

### 长期

- [ ] **Accessibility API 调研** — 评估 QQ 的 AX 树是否提供结构化数据
