NAME=360tools
BINDIR=bin
SOURCES=$(wildcard *.go)
BINARIES=${BINDIR}/${NAME}-darwin-amd64 ${BINDIR}/${NAME}-darwin-arm64 ${BINDIR}/${NAME}-darwin ${BINDIR}/${NAME}-linux-amd64 ${BINDIR}/${NAME}-linux-arm64 ${BINDIR}/${NAME}-linux-arm ${BINDIR}/${NAME}-windows.exe
TEMPLATES=$(wildcard *.template)

all: ${BINDIR} ${BINARIES}

${BINDIR}:
	mkdir -p ${BINDIR}
	
${BINDIR}/${NAME}-darwin-amd64: ${SOURCES} ${TEMPLATES}
	GOARCH=amd64 GOOS=darwin go build -o $@ ${SOURCES}

${BINDIR}/${NAME}-darwin-arm64: ${SOURCES} ${TEMPLATES}
	GOARCH=arm64 GOOS=darwin go build -o $@ ${SOURCES}

${BINDIR}/${NAME}-darwin: ${BINDIR}/${NAME}-darwin-amd64 ${BINDIR}/${NAME}-darwin-arm64
	makefat $@ $^

${BINDIR}/${NAME}-linux-amd64: ${SOURCES} ${TEMPLATES}
	GOARCH=amd64 GOOS=linux go build -o $@ ${SOURCES}

${BINDIR}/${NAME}-linux-arm64: ${SOURCES} ${TEMPLATES}
	GOARCH=arm64 GOOS=linux go build -o $@ ${SOURCES}

${BINDIR}/${NAME}-linux-arm: ${SOURCES} ${TEMPLATES}
	GOARCH=arm GOOS=linux go build -o $@ ${SOURCES}

${BINDIR}/${NAME}-windows.exe: ${SOURCES} ${TEMPLATES}
	GOARCH=amd64 GOOS=windows go build -o $@ ${SOURCES}

run:
	go run ${SOURCES}

clean:
	@go clean
	-@rm -rf ${BINDIR} 2>/dev/null || true
