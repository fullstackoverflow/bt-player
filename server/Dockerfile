# 构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /build

# 安装构建依赖
RUN apk add --no-cache git

# 复制 go mod 文件并下载依赖(利用 Docker 层缓存)
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用(使用构建缓存加速)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bt-player .

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装 ca-certificates (HTTPS 需要)
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 复制编译好的二进制文件
COPY --from=builder /build/bt-player .

# 运行应用
CMD ["./bt-player"]
