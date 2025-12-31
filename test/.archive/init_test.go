package test

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cfgstore"
	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
	"github.com/mikeschinkel/go-fsfix"
	"github.com/mikeschinkel/go-testutil"
	"github.com/mikeschinkel/gomion/gommod/gomcfg"
	"github.com/mikeschinkel/gomion/gommod/gompkg"
)

// TestInitRepos_ScanDirectory tests initializing repos by scanning a directory
func TestInitRepos_ScanDirectory(t *testing.T) {
	var tf *fsfix.RootFixture
	var repo1 *fsfix.RepoFixture
	var repo2 *fsfix.RepoFixture
	var result *gompkg.InitReposResult
	var err error
	var writer *testutil.BufferedWriter
	var logger *slog.Logger
	var store cfgstore.ConfigStore

	// Setup cfgstore logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfgstore.SetLogger(logger)

	// Create root fixture
	tf = fsfix.NewRootFixture("init-scan-test")
	defer tf.Cleanup()

	// Create first repo with go.mod at root
	repo1 = tf.AddRepoFixture(t, "repo1", nil)
	repo1.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo1

go 1.25.3
`,
	})

	// Create second repo with multiple modules
	repo2 = tf.AddRepoFixture(t, "repo2", nil)
	repo2.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo2

go 1.25.3
`,
	})

	// Add cmd submodule
	repo2cmd := repo2.AddDirFixture(t, "cmd", nil)
	repo2cmd.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo2/cmd

go 1.25.3
`,
	})

	// Create all fixtures
	tf.Create(t)

	// Create mock writer to capture output
	writer = testutil.NewBufferedWriter()

	// Create logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	// Initialize repos
	result, err = gompkg.Init(gompkg.InitArgs{
		FilePath: "",
		DirPath:  string(tf.Dir()),
		Writer:   writer,
		Logger:   logger,
	})

	if err != nil {
		t.Fatalf("InitializeRepos failed: %v", err)
	}

	// Debug output
	t.Logf("Writer output: %s", writer.GetPrintOutput())
	t.Logf("Writer error output: %s", writer.GetErrorOutput())

	if result.Initialized != 2 {
		t.Errorf("expected 2 initialized repos, got %d", result.Initialized)
	}

	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped repos, got %d", result.Skipped)
	}

	if len(result.Errors) > 0 {
		for i, e := range result.Errors {
			t.Logf("Error %d: %v", i, e)
		}
	}

	// Verify .gomion directory was created for repo1
	gomionPath := dt.DirPathJoin(repo1.Dir(), ".gomion")
	gomionExists, _ := gomionPath.Exists()
	if !gomionExists {
		t.Errorf(".gomion directory not created for repo1")
	}

	// Verify config file exists in repo1
	configPath := dt.FilepathJoin(gomionPath, "config.json")
	configExists, _ := configPath.Exists()
	if !configExists {
		t.Errorf("config.json not created in repo1/.gomion/")
	}

	// Verify .gomion directory was created for repo2
	gomionPath = dt.DirPathJoin(repo2.Dir(), ".gomion")
	gomionExists, _ = gomionPath.Exists()
	if !gomionExists {
		t.Errorf(".gomion directory not created for repo2")
	}

	// Load repo2 config using cfgstore
	store = gompkg.ProjectConfigStore(repo2.Dir())

	var repoConfig gomcfg.RepoConfig
	err = store.LoadJSON(&repoConfig)
	if err != nil {
		t.Fatalf("failed to load repo2 config: %v", err)
	}

	if len(repoConfig.Modules) != 2 {
		t.Errorf("expected 2 modules in repo2, got %d", len(repoConfig.Modules))
	}

	// Verify root module exists
	if _, ok := repoConfig.Modules["./"]; !ok {
		t.Errorf("root module not found in repo2 config")
	}

	// Verify cmd module exists
	if _, ok := repoConfig.Modules["./cmd"]; !ok {
		t.Errorf("cmd module not found in repo2 config")
	}

	// Verify cmd module has correct role
	if repoConfig.Modules["./cmd"].Kinds[0] != gompkg.ExeModuleKind {
		t.Errorf("expected cmd module role 'cli', got %q", repoConfig.Modules["./cmd"].Kinds[0])
	}
}

// TestInitRepos_ReadFile tests initializing repos from a file
func TestInitRepos_ReadFile(t *testing.T) {
	var tf *fsfix.RootFixture
	var repo1 *fsfix.RepoFixture
	var repo2 *fsfix.RepoFixture
	var inputFile *fsfix.FileFixture
	var result *gompkg.InitReposResult
	var err error
	var writer *mockWriter
	var logger *slog.Logger

	// Setup cfgstore logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfgstore.SetLogger(logger)

	// Create root fixture
	tf = fsfix.NewRootFixture("init-file-test")
	defer tf.Cleanup()

	// Create repos
	repo1 = tf.AddRepoFixture(t, "repo1", nil)
	repo1.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo1

go 1.25.3
`,
	})

	repo2 = tf.AddRepoFixture(t, "repo2", nil)
	repo2.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo2

go 1.25.3
`,
	})

	// Create input file with paths to go.mod files
	inputFile = tf.AddFileFixture(t, "modules.txt", &fsfix.FileFixtureArgs{
		Content: string(dt.FilepathJoin(repo1.Dir(), "go.mod")) + "\n" +
			string(dt.FilepathJoin(repo2.Dir(), "go.mod")) + "\n",
	})

	// Create all fixtures
	tf.Create(t)

	// Create mock writer
	writer = newMockWriter()

	// Create logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	// Initialize repos from file
	result, err = gompkg.InitializeRepos(&gompkg.InitializeRepoArgs{
		FilePath: string(inputFile.Filepath),
		DirArg:   "",
		AppInfo:  testAppInfo(),
		Options:  testOptions(),
		Writer:   writer,
		Logger:   logger,
	})

	if err != nil {
		t.Fatalf("InitializeRepos failed: %v", err)
	}

	if result.Initialized != 2 {
		t.Errorf("expected 2 initialized repos, got %d", result.Initialized)
	}

	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped repos, got %d", result.Skipped)
	}
}

// TestInitRepos_AlreadyManaged tests skipping already managed repos
func TestInitRepos_AlreadyManaged(t *testing.T) {
	var tf *fsfix.RootFixture
	var repo *fsfix.RepoFixture
	var gomionDir *fsfix.DirFixture
	var result *gompkg.InitReposResult
	var err error
	var writer *mockWriter
	var logger *slog.Logger

	// Setup cfgstore logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfgstore.SetLogger(logger)

	// Create root fixture
	tf = fsfix.NewRootFixture("init-already-managed-test")
	defer tf.Cleanup()

	// Create repo with existing .gomion/config.json
	repo = tf.AddRepoFixture(t, "repo", nil)
	repo.AddFileFixture(t, "go.mod", &fsfix.FileFixtureArgs{
		Content: `module github.com/test/repo

go 1.25.3
`,
	})

	// Create .gomion directory and config
	gomionDir = repo.AddDirFixture(t, ".gomion", nil)
	gomionDir.AddFileFixture(t, "config.json", &fsfix.FileFixtureArgs{
		Content: `{
  "modules": {
    "./": {
      "name": "repo",
      "role": ["lib"]
    }
  }
}`,
	})

	// Create all fixtures
	tf.Create(t)

	// Create mock writer
	writer = newMockWriter()

	// Create logger
	logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	// Try to initialize already managed repo
	result, err = gompkg.InitializeRepos(&gompkg.InitializeRepoArgs{
		FilePath: "",
		DirArg:   string(tf.Dir()),
		AppInfo:  testAppInfo(),
		Options:  testOptions(),
		Writer:   writer,
		Logger:   logger,
	})

	if err != nil {
		t.Fatalf("InitializeRepos failed: %v", err)
	}

	if result.Initialized != 0 {
		t.Errorf("expected 0 initialized repos, got %d", result.Initialized)
	}

	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped repo, got %d", result.Skipped)
	}

	// Verify output mentions skipping
	if !writer.ContainsOutput("Skipped") {
		t.Errorf("expected output to mention skipping, got: %s", writer.Output())
	}
}

// TestInferRole tests the inferRole function
func TestInferRole(t *testing.T) {
	tests := []struct {
		relPath  string
		expected string
	}{
		{"./", "lib"},
		{"./cmd", "cli"},
		{"./cmd/myapp", "cli"},
		{"./test", "test"},
		{"./tests", "test"},
		{"./pkg/mylib", "lib"},
		{"./internal/util", "lib"},
	}

	for _, tt := range tests {
		// We can't directly test inferRole since it's not exported,
		// but we can test it indirectly through InitializeRepos
		// For now, this is a placeholder for documentation
		t.Logf("Testing path %q expects role %q", tt.relPath, tt.expected)
	}
}

// mockWriter implements cliutil.Writer for testing
type mockWriter struct {
	buf    bytes.Buffer
	errBuf bytes.Buffer
}

func newMockWriter() *mockWriter {
	return &mockWriter{}
}

func (w *mockWriter) Printf(format string, args ...interface{}) {
	w.buf.WriteString(fmt.Sprintf(format, args...))
}

func (w *mockWriter) Errorf(format string, args ...interface{}) {
	w.errBuf.WriteString(fmt.Sprintf(format, args...))
}

func (w *mockWriter) Loud() cliutil.Writer {
	return w
}

func (w *mockWriter) V2() cliutil.Writer {
	return w
}

func (w *mockWriter) V3() cliutil.Writer {
	return w
}

func (w *mockWriter) Writer() io.Writer {
	return &w.buf
}

func (w *mockWriter) ErrWriter() io.Writer {
	return &w.errBuf
}

func (w *mockWriter) Output() string {
	return w.buf.String()
}

func (w *mockWriter) ErrorOutput() string {
	return w.errBuf.String()
}

func (w *mockWriter) ContainsOutput(substr string) bool {
	return strings.Contains(w.buf.String(), substr) || strings.Contains(w.errBuf.String(), substr)
}

// testAppInfo returns a test AppInfo
func testAppInfo() appinfo.AppInfo {
	return appinfo.New(appinfo.Args{
		Name:    "gomion",
		Version: "0.0.0-test",
	})
}

// testOptions returns test config options (nil is acceptable)
func testOptions() cfgstore.Options {
	return nil
}
