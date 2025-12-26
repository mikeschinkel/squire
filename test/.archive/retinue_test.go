package test

import (
	"testing"

	"github.com/mikeschinkel/go-fsfix"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModuleDiscovery tests basic module discovery functionality
func TestModuleDiscovery(t *testing.T) {
	t.Run("SingleModule", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet

		tf := fsfix.NewRootFixture("squiresvc-single-module")
		defer tf.Cleanup()

		// Create a simple project with .squire/config.json and go.mod
		pf := tf.AddRepoFixture(t, "test-project", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					".": {
						"name": "test-project",
						"role": ["lib"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/test-project

go 1.23
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")
		require.NotNil(t, ms, "Should return ModuleSet")

		assert.Len(t, ms.Modules, 1, "Should discover exactly one module")
		assert.Equal(t, "github.com/example/test-project", ms.Modules[0].ModulePath)
		assert.Equal(t, squiresvc.LibModuleKind, ms.Modules[0].Kind, "Root module should be detected as lib")
	})

	t.Run("MultipleModules", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet

		tf := fsfix.NewRootFixture("squiresvc-multiple-modules")
		defer tf.Cleanup()

		pf := tf.AddRepoFixture(t, "multi-module-project", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					".": {
						"name": "multi-module-project",
						"role": ["lib"]
					},
					"cmd": {
						"name": "cmd",
						"role": ["cmd"]
					},
					"lib": {
						"name": "lib",
						"role": ["lib"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/multi-module-project

go 1.23
`,
		})
		pf.AddFileFixture(t, "cmd/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/multi-module-project/cmd

go 1.23

require github.com/example/multi-module-project/lib v0.0.0
`,
		})
		pf.AddFileFixture(t, "lib/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/multi-module-project/lib

go 1.23
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")
		require.NotNil(t, ms, "Should return ModuleSet")

		assert.Len(t, ms.Modules, 3, "Should discover all three modules")

		modulesByPath := make(map[string]*squiresvc.Module)
		for _, m := range ms.Modules {
			modulesByPath[m.ModulePath] = m
		}

		assert.Contains(t, modulesByPath, "github.com/example/multi-module-project")
		assert.Contains(t, modulesByPath, "github.com/example/multi-module-project/cmd")
		assert.Contains(t, modulesByPath, "github.com/example/multi-module-project/lib")
	})

	t.Run("NoSquireConfig", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet

		tf := fsfix.NewRootFixture("squiresvc-no-config")
		defer tf.Cleanup()

		pf := tf.AddRepoFixture(t, "no-config-project", nil)
		pf.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/no-config

go 1.23
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))

		require.Error(t, err, "Should return error when no .squire config exists")
		assert.Nil(t, ms, "Should not return ModuleSet on error")
	})
}

// TestModuleOrdering tests dependency ordering functionality
func TestModuleOrdering(t *testing.T) {
	t.Run("SimpleLinearDependency", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet
		var ordered squiresvc.Modules

		tf := fsfix.NewRootFixture("squiresvc-linear-deps")
		defer tf.Cleanup()

		pf := tf.AddRepoFixture(t, "linear-project", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					"lib": {
						"name": "lib",
						"role": ["lib"]
					},
					"cmd": {
						"name": "cmd",
						"role": ["cmd"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "lib/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/linear/lib

go 1.23
`,
		})
		pf.AddFileFixture(t, "cmd/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/linear/cmd

go 1.23

require github.com/example/linear/lib v0.0.0
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")

		ordered, err = ms.OrderModules()
		require.NoError(t, err, "Should order modules without error")
		require.Len(t, ordered, 2, "Should return both modules")

		assert.Equal(t, "github.com/example/linear/lib", ordered[0].ModulePath,
			"lib should come first (no dependencies)")
		assert.Equal(t, "github.com/example/linear/cmd", ordered[1].ModulePath,
			"cmd should come second (depends on lib)")
	})

	t.Run("NoDependencies", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet
		var ordered squiresvc.Modules

		tf := fsfix.NewRootFixture("squiresvc-no-deps")
		defer tf.Cleanup()

		pf := tf.AddRepoFixture(t, "independent-modules", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					"module-a": {
						"name": "module-a",
						"role": ["lib"]
					},
					"module-b": {
						"name": "module-b",
						"role": ["lib"]
					},
					"module-c": {
						"name": "module-c",
						"role": ["lib"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "module-a/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/independent/module-a

go 1.23
`,
		})
		pf.AddFileFixture(t, "module-b/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/independent/module-b

go 1.23
`,
		})
		pf.AddFileFixture(t, "module-c/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/independent/module-c

go 1.23
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")

		ordered, err = ms.OrderModules()
		require.NoError(t, err, "Should order modules without error")
		require.Len(t, ordered, 3, "Should return all three modules")

		// All modules are independent, so any order is valid
		// Just verify we got all of them
		modulesByPath := make(map[string]bool)
		for _, m := range ordered {
			modulesByPath[m.ModulePath] = true
		}

		assert.True(t, modulesByPath["github.com/example/independent/module-a"])
		assert.True(t, modulesByPath["github.com/example/independent/module-b"])
		assert.True(t, modulesByPath["github.com/example/independent/module-c"])
	})

	t.Run("CircularDependency", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet
		var ordered squiresvc.Modules

		tf := fsfix.NewRootFixture("squiresvc-circular-deps")
		defer tf.Cleanup()

		pf := tf.AddRepoFixture(t, "circular-project", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					"module-a": {
						"name": "module-a",
						"role": ["lib"]
					},
					"module-b": {
						"name": "module-b",
						"role": ["lib"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "module-a/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/circular/module-a

go 1.23

require github.com/example/circular/module-b v0.0.0
`,
		})
		pf.AddFileFixture(t, "module-b/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/circular/module-b

go 1.23

require github.com/example/circular/module-a v0.0.0
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")

		ordered, err = ms.OrderModules()

		require.Error(t, err, "Should return error for circular dependency")
		assert.Contains(t, err.Error(), "cycle", "Error should mention cycle")
		assert.Nil(t, ordered, "Should not return ordered modules on cycle error")
	})

	t.Run("ComplexDependencyTree", func(t *testing.T) {
		var err error
		var ms *squiresvc.ModuleSet
		var ordered squiresvc.Modules

		tf := fsfix.NewRootFixture("squiresvc-complex-deps")
		defer tf.Cleanup()

		// Create a diamond dependency pattern:
		//      lib-base
		//      /      \
		//   lib-a    lib-b
		//      \      /
		//        cmd
		pf := tf.AddRepoFixture(t, "complex-project", nil)
		pf.AddFileFixture(t, ".squire/config.json", &fsfix.FileFixtureArgs{
			Content: `{
				"version": "1",
				"modules": {
					"lib-base": {
						"name": "lib-base",
						"role": ["lib"]
					},
					"lib-a": {
						"name": "lib-a",
						"role": ["lib"]
					},
					"lib-b": {
						"name": "lib-b",
						"role": ["lib"]
					},
					"cmd": {
						"name": "cmd",
						"role": ["cmd"]
					}
				}
			}`,
		})
		pf.AddFileFixture(t, "lib-base/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/complex/lib-base

go 1.23
`,
		})
		pf.AddFileFixture(t, "lib-a/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/complex/lib-a

go 1.23

require github.com/example/complex/lib-base v0.0.0
`,
		})
		pf.AddFileFixture(t, "lib-b/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/complex/lib-b

go 1.23

require github.com/example/complex/lib-base v0.0.0
`,
		})
		pf.AddFileFixture(t, "cmd/go.mod", &fsfix.FileFixtureArgs{
			Content: `module github.com/example/complex/cmd

go 1.23

require (
	github.com/example/complex/lib-a v0.0.0
	github.com/example/complex/lib-b v0.0.0
)
`,
		})

		tf.Create(t)

		ms, err = squiresvc.DiscoverModules(string(pf.Dir()))
		require.NoError(t, err, "Should discover modules without error")

		ordered, err = ms.OrderModules()
		require.NoError(t, err, "Should order modules without error")
		require.Len(t, ordered, 4, "Should return all four modules")

		// Verify dependency order constraints
		moduleOrder := make(map[string]int)
		for i, m := range ordered {
			moduleOrder[m.ModulePath] = i
		}

		// lib-base must come before lib-a and lib-b
		assert.Less(t, moduleOrder["github.com/example/complex/lib-base"],
			moduleOrder["github.com/example/complex/lib-a"],
			"lib-base should come before lib-a")
		assert.Less(t, moduleOrder["github.com/example/complex/lib-base"],
			moduleOrder["github.com/example/complex/lib-b"],
			"lib-base should come before lib-b")

		// Both lib-a and lib-b must come before cmd
		assert.Less(t, moduleOrder["github.com/example/complex/lib-a"],
			moduleOrder["github.com/example/complex/cmd"],
			"lib-a should come before cmd")
		assert.Less(t, moduleOrder["github.com/example/complex/lib-b"],
			moduleOrder["github.com/example/complex/cmd"],
			"lib-b should come before cmd")
	})
}
