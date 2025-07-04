# Build
FROM golang:1.24.4 as build_context

ENV OUTDIR=/out
RUN mkdir -p ${OUTDIR}/usr/local/bin/

WORKDIR /workspace
ADD . /workspace/

RUN bash quick_build.sh
RUN mv simple-one-api ${OUTDIR}/usr/local/bin/

# 使用一个轻量级的基础镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 通过构建参数选择架构
ARG ARCH=amd64
COPY --from=build_context /out/usr/local/bin/simple-one-api /app/simple-one-api

# 复制当前目录的static目录内的内容到镜像中
COPY static /app/static

# 暴露应用运行的端口（假设为9090）
EXPOSE 9090

# 运行可执行文件
CMD ["sh"]
