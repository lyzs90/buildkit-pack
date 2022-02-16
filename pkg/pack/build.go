package pack

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	"golang.org/x/sync/errgroup"
)

const (
	localNameContext  = "context"
	buildArgPrefix    = "build-arg:"
	keyBuilder = "builder"
)

// Build using the pack cli
func Build(ctx context.Context, c client.Client) (*client.Result, error) {
	opts := c.BuildOpts().Opts

	// also accept build args from BuildKit
	for k, v := range opts {
		if strings.HasPrefix(k, buildArgPrefix) {
			opts[strings.TrimPrefix(k, buildArgPrefix)] = v
		}
	}

	builder := ""
	if v, ok := opts[keyBuilder]; ok {
		builder = v
	}

	// TODO: support git/http sources
	src := llb.Local(localNameContext, llb.SessionID(c.BuildOpts().SessionID), llb.SharedKeyHint("pack-src"))

	// don't assume builder image has pack cli
	img := llb.Image("alpine").Run(llb.Shlex("apk add --no-cache curl"), llb.WithCustomName("install curl"))
	img = img.Run(llb.Shlex(`(curl -sSL "https://github.com/buildpacks/pack/releases/download/v0.24.0/pack-v0.24.0-linux.tgz" | sudo tar -C /usr/local/bin/ --no-same-owner -xzv pack)`), llb.WithCustomName("install pack cli"))
	st := runBuilder(c, img, fmt.Sprintf(`/usr/local/bin/pack build --builder %s --path %s`, builder, "/workspace"), llb.Dir("/workspace"))
	st.AddMount("/workspace", src, llb.Readonly)
	st.AddMount("/tmp", llb.Scratch(), llb.AsPersistentCacheDir("buildpack-build-cache", llb.CacheMountShared))

	def, err := st.Marshal(ctx, llb.WithCaps(c.BuildOpts().LLBCaps))
	if err != nil {
		return nil, err
	}

	eg, ctx := errgroup.WithContext(ctx)

	var res *client.Result
	eg.Go(func() error {
		r, err := c.Solve(ctx, client.SolveRequest{
			Definition: def.ToPB(),
		})
		if err != nil {
			return err
		}
		res = r
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return res, nil
}

func runBuilder(c client.Client, img llb.ExecState, cmd string, opts ...llb.RunOption) llb.ExecState {
	// TODO: check if this is still required	
	// work around docker 18.06 executor with no cgroups mounted because build has
	// a hard requirement on the file

	caps := c.BuildOpts().LLBCaps

	mountCgroups := (&caps).Supports(pb.CapExecCgroupsMounted) != nil

	opts = append(opts, llb.WithCustomName(cmd))

	if mountCgroups {
		cmd = `sh -c "mkdir -p /sys/fs/cgroup/memory && echo 9223372036854771712 > /sys/fs/cgroup/memory/memory.limit_in_bytes && ` + cmd + `"`
	}

	es := img.Run(append(opts, llb.Shlex(cmd))...)

	if mountCgroups {
		es.AddMount("/sys/fs/cgroup", llb.Scratch())
		alpine := llb.Image("alpine").Run(llb.Shlex(`sh -c 'echo "127.0.0.1 $(hostname)" > /out/hosts'`), llb.WithCustomName("[internal] make hostname resolvable"))
		hosts := alpine.AddMount("/out", llb.Scratch())
		es.AddMount("/etc/hosts", hosts, llb.SourcePath("hosts"), llb.Readonly)
	}

	return es
}
