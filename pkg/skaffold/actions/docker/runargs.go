/*
Copyright 2026 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
)

// RunArgs is the parsed, whitelisted projection of a user-supplied
// customActions.*.executionMode.local.runArgs list.
//
// Only a small, deliberately conservative subset of `docker run` flags is
// recognised. Unknown flags are rejected so users fail fast instead of
// silently being ignored. See ParseRunArgs for the full list.
type RunArgs struct {
	NetworkMode string
	Binds       []string
	Env         []string
	User        string
	ExtraHosts  []string
	Tmpfs       map[string]string
	Privileged  bool
	CapAdd      []string
	CapDrop     []string
}

// ParseRunArgs parses a docker-run-style argument list and returns a
// whitelisted RunArgs projection. Supported flags:
//
//	--network=VALUE
//	-v=SRC:DST[:MODE]          (also --volume=...)
//	-e=KEY=VALUE               (also --env=...)
//	--user=UID[:GID]
//	--add-host=HOST:IP
//	--tmpfs=PATH[:OPTIONS]
//	--privileged
//	--cap-add=CAP
//	--cap-drop=CAP
//
// Each flag must be in the `--flag=value` (or `-f=value`) form; the
// space-separated variant is not supported to keep the parser unambiguous.
// A nil / empty input returns a nil *RunArgs.
func ParseRunArgs(args []string) (*RunArgs, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := &RunArgs{}
	for i, raw := range args {
		arg := strings.TrimSpace(raw)
		if arg == "" {
			continue
		}
		if arg == "--privileged" || arg == "--privileged=true" {
			out.Privileged = true
			continue
		}
		if arg == "--privileged=false" {
			out.Privileged = false
			continue
		}
		key, val, ok := strings.Cut(arg, "=")
		if !ok {
			// No '=' means either an unknown bare flag or the space-separated form.
			if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("runArgs[%d] %q: unsupported flag %q (only --flag=value form is supported; allowed: --network, -v/--volume, -e/--env, --user, --add-host, --tmpfs, --privileged, --cap-add, --cap-drop)", i, raw, arg)
			}
			return nil, fmt.Errorf("runArgs[%d] %q: only --flag=value form is supported (no space-separated values)", i, raw)
		}
		switch key {
		case "--network":
			out.NetworkMode = val
		case "-v", "--volume":
			out.Binds = append(out.Binds, val)
		case "-e", "--env":
			out.Env = append(out.Env, val)
		case "--user":
			out.User = val
		case "--add-host":
			out.ExtraHosts = append(out.ExtraHosts, val)
		case "--tmpfs":
			if out.Tmpfs == nil {
				out.Tmpfs = map[string]string{}
			}
			mountPath, opts, _ := strings.Cut(val, ":")
			out.Tmpfs[mountPath] = opts
		case "--cap-add":
			out.CapAdd = append(out.CapAdd, val)
		case "--cap-drop":
			out.CapDrop = append(out.CapDrop, val)
		default:
			return nil, fmt.Errorf("runArgs[%d] %q: unsupported flag %q (allowed: --network, -v/--volume, -e/--env, --user, --add-host, --tmpfs, --privileged, --cap-add, --cap-drop)", i, raw, key)
		}
	}
	return out, nil
}

// ApplyToContainerConfig overlays parsed runArgs fields that belong on the
// container.Config (User, additional env vars) onto cfg in place. A nil
// receiver is a no-op.
func (r *RunArgs) ApplyToContainerConfig(cfg *container.Config) {
	if r == nil || cfg == nil {
		return
	}
	if r.User != "" {
		cfg.User = r.User
	}
	if len(r.Env) > 0 {
		// RunArgs env entries win over anything already on the container
		// config — consistent with deploy-parameter precedence.
		cfg.Env = append(cfg.Env, r.Env...)
	}
}

// ApplyToHostConfig overlays parsed runArgs fields that belong on the
// container.HostConfig onto hc in place. NetworkMode is only overridden
// when the user provided one. A nil receiver is a no-op.
func (r *RunArgs) ApplyToHostConfig(hc *container.HostConfig) {
	if r == nil || hc == nil {
		return
	}
	if r.NetworkMode != "" {
		hc.NetworkMode = container.NetworkMode(r.NetworkMode)
	}
	if len(r.Binds) > 0 {
		hc.Binds = append(hc.Binds, r.Binds...)
	}
	if len(r.ExtraHosts) > 0 {
		hc.ExtraHosts = append(hc.ExtraHosts, r.ExtraHosts...)
	}
	if len(r.Tmpfs) > 0 {
		if hc.Tmpfs == nil {
			hc.Tmpfs = map[string]string{}
		}
		for k, v := range r.Tmpfs {
			hc.Tmpfs[k] = v
		}
	}
	if r.Privileged {
		hc.Privileged = true
	}
	if len(r.CapAdd) > 0 {
		hc.CapAdd = append(hc.CapAdd, r.CapAdd...)
	}
	if len(r.CapDrop) > 0 {
		hc.CapDrop = append(hc.CapDrop, r.CapDrop...)
	}
}
