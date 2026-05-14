# SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

# this dockerfile hopes to collect the correct dependencies and build b0gus without any error.
FROM golang:1.26 AS builder-env
# <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<< stage-1
ARG antlr4Tag=4.13.2
ARG antlr4Path="/usr/local/bin/antlr4"
ARG sourcePath="/etc/apt/sources.list.d/debian.sources"

# for other country/region, set region to the faster mirror source in command line by
# passing arguments like: --build-arg region="us"
ARG region="cn"
ARG arch="amd64"
ARG osType="linux"

WORKDIR /b0gus-builder
COPY . .

# protoc, antlr4 (>= 4.13.2) is required for building.
# Here, this dockerfile will help to build antlr4 from its source and install to /usr/local/bin
RUN chmod -R +x /b0gus-builder && cp $sourcePath $sourcePath.bak &&                                        \
    sed -i "s|deb.debian.org|ftp.$region.debian.org|g" $sourcePath &&                                      \
    apt update && apt install -y protoc-gen-go wget make binutils gcc openjdk-25-jdk maven git &&          \
    git clone -b $antlr4Tag --single-branch --depth=1 https://github.com/antlr/antlr4.git &&               \
    cd antlr4 && export MAVEN_OPTS="-Xmx1G" && mvn install -DskipTests && cd .. &&                         \
    mkdir -p /usr/local/share/ && cp -R antlr4 /usr/local/share/ &&                                        \
    echo "#!/usr/bin/env bash" >> $antlr4Path &&                                                           \
    echo "antlr4Dir=/usr/local/share/antlr4" >> $antlr4Path &&                                             \
    echo "classPath=\"\\" >> $antlr4Path &&                                                                \
    echo "\$antlr4Dir/tool/target/antlr4-$antlr4Tag.jar:\\" >> $antlr4Path &&                              \
    echo "\$antlr4Dir/tool/target/antlr4-$antlr4Tag-complete.jar:\\" >> $antlr4Path &&                     \
    echo "\$antlr4Dir/antlr4-maven-plugin/target/antlr4-maven-plugin-$antlr4Tag.jar:\\" >> $antlr4Path &&  \
    echo "\$antlr4Dir/runtime/Java/target/antlr4-runtime-$antlr4Tag.jar:\"" >> $antlr4Path &&              \
    echo "exec java -cp \$classPath org.antlr.v4.Tool \"\$@\"" >> $antlr4Path && chmod a+rx $antlr4Path && \
	make release Arch=$arch osType=$osType

FROM alpine:latest AS builder
# <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<< stage-2
WORKDIR /app
# then we have b0gus in the app directory.
COPY --from=builder-env /b0gus-builder/b0gus /app/b0gus
# docker run --rm --entrypoint /bin/cat b0gus-image /app/b0gus > ./b0gus-exe
