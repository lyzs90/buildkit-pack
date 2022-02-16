package main

import (
	"github.com/lyzs90/buildkit-pack/pkg/pack"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := grpcclient.RunFromEnvironment(appcontext.Context(), pack.Build); err != nil {
		logrus.Errorf("fatal error: %+v", err)
		panic(err)
	}
}
