FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制Go模块定义
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pod_connect_exporter ./cmd/exporter

# 使用最小镜像
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/pod_connect_exporter .

# 设置时区
ENV TZ=Asia/Shanghai

# 声明端口
EXPOSE 28880

# 运行应用
ENTRYPOINT ["/app/pod_connect_exporter"] 