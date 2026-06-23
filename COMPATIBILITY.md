# 兼容性记录

## 已验证版本

| 日期 | QQ 版本 | Build | macOS | 安装路径 | Bundle ID |
|---|---|---|---|---|---|
| 2026-06-23 | 6.9.87 | 36368 | 11.x (Big Sur) | `/Applications/QQ 2.app/` | com.tencent.qq |

## 版本检测方法

```bash
osascript -e 'tell application "Finder" to get version of application "QQ"'
# 或:
plutil -p "/Applications/QQ 2.app/Contents/Info.plist" | grep CFBundleShortVersionString
```

## 注意

- QQ 基于 Electron，不同版本聊天窗口布局可能变化
- 坐标聚类阈值 `yThreshold = 0.025` 可能需要随版本调整
- 字段推断规则基于窗口布局特征，版本更新后需验证
