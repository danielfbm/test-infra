FROM docker:19.03-dind

RUN apk add --no-cache \
                ca-certificates

ENV GOLANG_VERSION 1.12.12

RUN set -eux; \
        apk add --no-cache --virtual .build-deps \
                bash \
                gcc \
                musl-dev \
                openssl \
                go \
        ; \
        export \
# set GOROOT_BOOTSTRAP such that we can actually build Go
                GOROOT_BOOTSTRAP="$(go env GOROOT)" \
# ... and set "cross-building" related vars to the installed system's values so that we create a build targeting the proper arch
# (for example, if our build host is GOARCH=amd64, but our build env/image is GOARCH=386, our build needs GOARCH=386)
                GOOS="$(go env GOOS)" \
                GOARCH="$(go env GOARCH)" \
                GOHOSTOS="$(go env GOHOSTOS)" \
                GOHOSTARCH="$(go env GOHOSTARCH)" \
        ; \
# also explicitly set GO386 and GOARM if appropriate
# https://github.com/docker-library/golang/issues/184
        apkArch="$(apk --print-arch)"; \
        case "$apkArch" in \
                armhf) export GOARM='6' ;; \
                x86) export GO386='387' ;; \
        esac; \
        \
        wget -O go.tgz "https://golang.org/dl/go$GOLANG_VERSION.src.tar.gz"; \
        echo '6d7a5ba05476609a7614af3292f29c3be06327503c1f1fdc02ef417870fd6926 *go.tgz' | sha256sum -c -; \
        tar -C /usr/local -xzf go.tgz; \
        rm go.tgz; \
        \
        cd /usr/local/go/src; \
        ./make.bash; \
        \
        rm -rf \
# https://github.com/golang/go/blob/0b30cf534a03618162d3015c8705dd2231e34703/src/cmd/dist/buildtool.go#L121-L125
                /usr/local/go/pkg/bootstrap \
# https://golang.org/cl/82095
# https://github.com/golang/build/blob/e3fe1605c30f6a3fd136b561569933312ede8782/cmd/release/releaselet.go#L56
                /usr/local/go/pkg/obj \
        ; \
        apk del .build-deps; \
        \
        export PATH="/usr/local/go/bin:$PATH"; \
        go version

RUN apk add --no-cache make curl git gcc libc-dev

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
WORKDIR $GOPATH

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
    chmod +x kubectl && \
    mv kubectl /go/bin/ && \
    wget https://github.com/kubernetes-sigs/kind/releases/download/v0.6.1/kind-linux-amd64 && \
    chmod +x kind-linux-amd64 && \
    mv kind-linux-amd64 /go/bin/kind