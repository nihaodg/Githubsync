# GitHub Sync 同步工具

一个简洁的 GitHub 仓库同步工具，无需安装 Git 即可管理你的 GitHub 仓库。

## 功能特点

- **Clone 仓库** - 输入 GitHub 仓库地址，一键克隆到本地
- **Pull 拉取** - 从远程仓库拉取最新代码
- **Push 推送** - 将本地提交推送到 GitHub
- **Commit 提交** - 记录本地修改
- **查看状态** - 实时查看文件修改状态
- **提交历史** - 查看仓库的提交记录

## 使用方法

### 1. 配置 Token

首次使用需要输入 GitHub Personal Access Token (PAT)：
1. 打开 https://github.com/settings/tokens
2. 点击 "Generate new token (classic)"
3. 勾选 `repo` 权限
4. 生成后复制 Token
5. 在软件中粘贴 Token 并保存

### 2. 克隆仓库

在软件中输入 GitHub 仓库地址，例如：
```
https://github.com/user/repo
```

### 3. 同步操作

- **拉取 (Pull)** - 获取远程最新代码
- **推送 (Push)** - 上传本地提交到 GitHub
- **提交 (Commit)** - 填写提交说明，保存本地修改

## 技术架构

- **后端**: Go 语言
- **前端**: React + TypeScript
- **框架**: Wails 2.x (跨平台桌面应用框架)
- **Git 操作**: go-git (纯 Go 实现的 Git 客户端，无需系统 Git)

## 系统要求

- Windows 10/11 (64位)
- 不需要安装 Git

## 下载使用

直接下载 `GitHubSync.exe` 运行即可：
- [GitHubSync.exe](https://github.com/nihaodg/Githubsync/releases/latest/download/GitHubSync.exe)

## 注意事项

- Token 只存储在本地配置文件，不会上传到任何服务器
- 仓库默认存储在 `C:\Users\你的用户名\GitHubSync\repos`
- 首次使用前请确保网络连接正常

## 许可证

MIT License
