#!/bin/bash
export GLIDE_TAG='v0.13.2'
export GOMETALINTER_TAG='v2.0.11'
 go get -v github.com/alecthomas/gometalinter         && \
cd $GOPATH/src/github.com/alecthomas/gometalinter    && \
git checkout $GOMETALINTER_TAG                       && \
go install                                           && \
gometalinter --install
cd -
 # go get -u honnef.co/go/tools/... ?
go get -v github.com/Masterminds/glide               && \
cd $GOPATH/src/github.com/Masterminds/glide          && \
git checkout $GLIDE_TAG                              && \
make build                                           && \
mv glide `which glide`                               && \
cd -