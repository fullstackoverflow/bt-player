# BT Player

BT 流式播放器 - 支持边下边播

## 📺 演示视频

## 功能特性

- ✅ BT/磁力链接流式播放
- ✅ 支持 SRT/VTT/ASS 字幕上传
- ✅ 浏览器插件快捷播放
- ✅ Docker 一键部署

## 快速开始

### Docker 部署

```bash
docker run -d \
  --name bt-player \
  -p 8080:8080 \
  -v bt-data:/app/torrent-data \
  ghcr.io/fullstackoverflow/bt-player:latest
```

## 使用说明

### 1. 启动服务

服务启动后访问: http://localhost:8080/assets/player.html

### 2. 安装浏览器插件

1. 下载插件 ZIP 文件
2. 解压到固定位置
3. 打开 Chrome `chrome://extensions/`
4. 开启"开发者模式"
5. 点击"加载已解压的扩展程序"
6. 选择解压后的文件夹

### 3. 配置插件

1. 点击插件图标
2. 设置服务器地址: `http://localhost:8080`
3. 保存设置

## 端口说明

- `8080`: HTTP 服务端口 (可修改)

## 常见问题

### Q: 视频加载慢?
A: 取决于种子健康度和网络速度,一个种子获取元信息超过2分钟会停止尝试

### Q: 右键菜单不显示?
A: 确认插件已启用且配置了服务器地址

### Q: 怎么使用字幕?
A: 播放器页面设置立即生效,支持 SRT/VTT/ASS 格式

### Q: Docker 容器无法访问?
A: 检查端口映射 `-p 8080:8080` 是否正确
