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

COPY --from=build_context /out/usr/local/bin/simple-one-api /app/simple-one-api

# 复制当前目录的static目录内的内容到镜像中
COPY static /app/static

# 创建非 root 用户和组
RUN addgroup -S edgewize && adduser -S edgewize -G edgewize

# 创建应用目录
RUN chown -R edgewize:edgewize /app/simple-one-api

USER edgewize

# 暴露应用运行的端口（假设为9090）
EXPOSE 9090

# 运行可执行文件
CMD ["sh"]
