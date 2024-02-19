package manager

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
        "os"
        "fmt"
        "sort"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var pluginNameRe = regexp.MustCompile("^[a-z][a-z0-9]*$")

// Plugin represents a potential plugin with all it's metadata.
type Plugin struct {
	Metadata

	Name string `json:",omitempty"`
	Path string `json:",omitempty"`

	// Err is non-nil if the plugin failed one of the candidate tests.
	Err error `json:",omitempty"`

	// ShadowedPaths contains the paths of any other plugins which this plugin takes precedence over.
	ShadowedPaths []string `json:",omitempty"`
}
func writeCoverageToFile(filename string, data map[int]bool) {
    f, err := os.Create(filename)
    if err != nil {
        panic(err)
    }
    defer f.Close()
    
    var builder strings.Builder
    var sum float64
    type keyValue struct {
        key int
        val bool
    }
    
    var kv []keyValue
    
    for k, v := range data {
        if v {
            sum++
        }
        kv = append(kv, keyValue{k, v})
    }
    
    sort.Slice(kv, func(i, j int) bool {
        return kv[i].key < kv[j].key
    })
    
    for _, v := range kv {
        builder.WriteString(fmt.Sprintf("Branch %d\t -\t %t\n", v.key, v.val))
    }
    
    builder.WriteString(fmt.Sprintf("\nCoverage Percentage - %2.f%%", (sum / float64(len(data)))*100))
    
    f.WriteString(builder.String())
}

var branch_coverage_flags = map[int]bool {
    1:  false,
    2:  false,
    3:  false,
    4:  false,
    5:  false,
    6:  false,
    7:  false,
    8:  false,
    9:  false,
    10: false,
    11: false,
    12: false,
    13: false,
    14: false,
    15: false,
    16: false,
    17: false,
    18: false,
    19: false,
    20: false,
    21: false,
    22: false,
    23: false,
    24: false,
    25: false,
}
// newPlugin determines if the given candidate is valid and returns a
// Plugin.  If the candidate fails one of the tests then `Plugin.Err`
// is set, and is always a `pluginError`, but the `Plugin` is still
// returned with no error. An error is only returned due to a
// non-recoverable error.
func newPlugin(c Candidate, cmds []*cobra.Command) (Plugin, error) {
	path := c.Path()
	if path == "" {
		branch_coverage_flags[1] = true
		return Plugin{}, errors.New("plugin candidate path cannot be empty")
	} else {
		branch_coverage_flags[2] = true
	}

	// The candidate listing process should have skipped anything
	// which would fail here, so there are all real errors.
	fullname := filepath.Base(path)
	if fullname == "." {
		branch_coverage_flags[3] = true
		return Plugin{}, errors.Errorf("unable to determine basename of plugin candidate %q", path)
	} else {
		branch_coverage_flags[4] = true
	}

	var err error
	if fullname, err = trimExeSuffix(fullname); err != nil {
		branch_coverage_flags[5] = true
		return Plugin{}, errors.Wrapf(err, "plugin candidate %q", path)
	} else {
		branch_coverage_flags[6] = true
	}
	if !strings.HasPrefix(fullname, NamePrefix) {
		branch_coverage_flags[7] = true
		return Plugin{}, errors.Errorf("plugin candidate %q: does not have %q prefix", path, NamePrefix)
	} else {
		branch_coverage_flags[8] = true
	}

	p := Plugin{
		Name: strings.TrimPrefix(fullname, NamePrefix),
		Path: path,
	}

	// Now apply the candidate tests, so these update p.Err.
	if !pluginNameRe.MatchString(p.Name) {
		branch_coverage_flags[9] = true
		p.Err = NewPluginError("plugin candidate %q did not match %q", p.Name, pluginNameRe.String())
		return p, nil
	} else {
		branch_coverage_flags[10] = true
	}

	for _, cmd := range cmds {
		branch_coverage_flags[11] = true
		// Ignore conflicts with commands which are
		// just plugin stubs (i.e. from a previous
		// call to AddPluginCommandStubs).
		if IsPluginCommand(cmd) {
			branch_coverage_flags[12] = true
			continue
		} else {
			branch_coverage_flags[13] = true
		}
		if cmd.Name() == p.Name {
			branch_coverage_flags[14] = true
			p.Err = NewPluginError("plugin %q duplicates builtin command", p.Name)
			return p, nil
		} else {
			branch_coverage_flags[15] = true
		}
		if cmd.HasAlias(p.Name) {
			branch_coverage_flags[16] = true
			p.Err = NewPluginError("plugin %q duplicates an alias of builtin command %q", p.Name, cmd.Name())
			return p, nil
		} else {
			branch_coverage_flags[17] = true
		}
	}

	// We are supposed to check for relevant execute permissions here. Instead we rely on an attempt to execute.
	meta, err := c.Metadata()
	if err != nil {
		branch_coverage_flags[18] = true
		p.Err = wrapAsPluginError(err, "failed to fetch metadata")
		return p, nil
	} else {
		branch_coverage_flags[19] = true
	}

	if err := json.Unmarshal(meta, &p.Metadata); err != nil {
		branch_coverage_flags[20] = true
		p.Err = wrapAsPluginError(err, "invalid metadata")
		return p, nil
	} else {
		branch_coverage_flags[21] = true
	}
	if p.Metadata.SchemaVersion != "0.1.0" {
		branch_coverage_flags[22] = true
		p.Err = NewPluginError("plugin SchemaVersion %q is not valid, must be 0.1.0", p.Metadata.SchemaVersion)
		return p, nil
	} else {
		branch_coverage_flags[23] = true
	}
	if p.Metadata.Vendor == "" {
		branch_coverage_flags[24] = true
		p.Err = NewPluginError("plugin metadata does not define a vendor")
		return p, nil
	} else {
		branch_coverage_flags[25] = true
	}
	return p, nil
}
