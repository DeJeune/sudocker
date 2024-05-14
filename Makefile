Version=$(shell git tag --sort=committerdate | tail -n 1)
GoVersion=$(shell go version)
BuildTime=$(shell date +"%F %T")
GitCommit=$(shell git rev-parse HEAD)

LDFLAGS=-ldflags "-X 'github.com/DeJeune/sudocker/cli/version.Version=${Version}' \
-X 'github.com/DeJeune/sudocker/cli/version.GoVersion=${GoVersion}' \
-X 'github.com/DeJeune/sudocker/cli/version.GoVersion=${GoVersion}' \
-X 'github.com/DeJeune/sudocker/cli/version.BuildTime=${BuildTime}' \
-X 'github.com/DeJeune/sudocker/cli/version.GitCommit=${GitCommit}'"


build:
	go build ${LDFLAGS} -o sudocker

clean:
	rm -f sudocker
.PHONY: build clean