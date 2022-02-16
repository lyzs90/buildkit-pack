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
	img := llb.Image("alpine").
		Run(llb.Shlex("apk add --no-cache curl"), llb.WithCustomName("installing curl")).
		Run(llb.Shlex(`curl -sSL https://github.com/buildpacks/pack/releases/download/v0.24.0/pack-v0.24.0-linux.tgz -O`), llb.WithCustomName("fetching pack cli")).
		Run(llb.Shlex(`tar -zxvf pack-v0.24.0-linux.tgz`), llb.WithCustomName("unzipping pack cli")).
		Run(llb.Shlex(`mv pack /usr/local/bin/pack`), llb.WithCustomName("installing pack cli"))
	build := runBuilder(c, img, fmt.Sprintf(`/usr/local/bin/pack build %s --builder %s --path %s`, "temp", builder, "/workspace"), llb.Dir("/workspace"))
	build.AddMount("/workspace", src, llb.Readonly)
	build.AddMount("/tmp", llb.Scratch(), llb.AsPersistentCacheDir("buildpack-build-cache", llb.CacheMountShared))
	
	// TODO: offer two routes: direct to registry via pack CLI. but then what about buildkit?
	// what purpose did buildkit serve as there's nothing to parallelize

	// TODO: for the second case, export temp that was built by pack cli. output is an image, not a directory...
	build.Run(llb.Shlex("docker save temp > /out/temp.tar"))
	// FIXME: this should be the dest image i.e. the run image
	extract := llb.Image("alpine").Run(llb.Shlex(``), llb.WithCustomName("copy temp image to stack"), llb.Dir("/in"))
	extract.AddMount("/in", build.Root(), llb.SourcePath("out"), llb.Readonly)
	st := extract.AddMount("/out", llb.Image(runName))

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
