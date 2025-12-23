package gitutils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mikeschinkel/go-dt"
	"golang.org/x/mod/semver"
)

type GitRef string

type Repo struct {
	Root   dt.DirPath
	Branch GitRef
	Remote GitRemote
}

func Open(dir dt.DirPath) (repo *Repo, err error) {
	var root dt.DirPath
	var out []byte

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = string(dir)
	out, err = cmd.Output()
	if err != nil {
		err = NewErr(ErrNotGitRepo, err)
		goto end
	}

	root = dt.DirPath(bytes.TrimSpace(out))
	if root == "" {
		err = ErrNotGitRepo
	}
	repo = &Repo{
		Root: root,
	}

	// Get current branch
	repo.Branch, err = repo.currentBranch()
	if err != nil {
		goto end
	}

	// Get Remote name
	repo.Remote, err = repo.currentRemote()
	if err != nil {
		goto end
	}

end:
	if err != nil {
		err = WithErr(err, dir.ErrKV())
	}
	return repo, err
}

func (r *Repo) RevParse(ref string) (string, error) {
	out, err := r.runGit(context.Background(), r.Root, "rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r *Repo) IsDirty() (bool, error) {
	out, err := r.runGit(context.Background(), r.Root, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// IsDirtyInPath checks if there are uncommitted changes within a specific path
func (r *Repo) IsDirtyInPath(relPath dt.PathSegments) (bool, error) {
	out, err := r.runGit(context.Background(), r.Root, "status", "--porcelain", string(relPath))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Repo) UpstreamState() (us UpstreamState, err error) {
	var out string
	var fields []string

	_, err = r.runGit(context.Background(), r.Root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		goto end
	}
	out, err = r.runGit(context.Background(), r.Root, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		goto end
	}
	fields = strings.Fields(strings.TrimSpace(out))
	if len(fields) != 2 {
		err = fmt.Errorf("unexpected rev-list output: %q", out)
		goto end
	}
	us = UpstreamState{
		ahead:  r.toIntPtr(fields[0]),
		behind: r.toIntPtr(fields[1]),
	}
end:
	return us, err
}

// currentBranch returns the name of the current branch
func (r *Repo) currentBranch() (ref GitRef, err error) {
	var branch string
	branch, err = r.runGit(context.Background(), r.Root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		goto end
	}
	ref = GitRef(strings.TrimSpace(branch))
end:
	return ref, err
}

type RemoteName string
type GitRemote struct {
	Name   RemoteName
	Branch GitRef
}

func (r GitRemote) String() string {
	return fmt.Sprintf("%s/%s", r.Name, r.Branch)
}

func (r GitRemote) IsValid() bool {
	return len(r.Name) > 0 && len(r.Branch) > 0
}

// currentRenote the name of the upstream Remote (typically "origin")
func (r *Repo) currentRemote() (remote GitRemote, err error) {
	var out string
	var parts []string

	out, err = r.runGit(context.Background(), r.Root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		goto end
	}
	remote = GitRemote{}
	// Output is like "origin/main", extract "origin"
	parts = strings.Split(strings.TrimSpace(out), "/")
	switch len(parts) {
	case 1:
		remote.Name = RemoteName(parts[0])
		fallthrough
	case 2:
		remote.Branch = GitRef(parts[1])
	}
end:
	return remote, err
}

// StatusCounts returns counts of staged, unstaged, and untracked files
type StatusCounts struct {
	Staged    int
	Unstaged  int
	Untracked int
}

func (r *Repo) StatusCounts() (counts StatusCounts, err error) {
	var out string
	var lines []string

	out, err = r.runGit(context.Background(), r.Root, "status", "--porcelain")
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) < 2 {
			continue
		}
		x := line[0] // Index (staged) status
		y := line[1] // Worktree (unstaged) status

		// Untracked files
		if x == '?' && y == '?' {
			counts.Untracked++
			continue
		}

		// Staged changes
		if x != ' ' && x != '?' {
			counts.Staged++
		}

		// Unstaged changes
		if y != ' ' && y != '?' {
			counts.Unstaged++
		}
	}

end:
	return counts, err
}

// StatusCountsInPath returns counts of staged, unstaged, and untracked files within a specific path
func (r *Repo) StatusCountsInPath(relPath dt.PathSegments) (counts StatusCounts, err error) {
	var out string
	var lines []string

	out, err = r.runGit(context.Background(), r.Root, "status", "--porcelain", string(relPath))
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if len(line) < 2 {
			continue
		}
		x := line[0] // Index (staged) status
		y := line[1] // Worktree (unstaged) status

		// Untracked files
		if x == '?' && y == '?' {
			counts.Untracked++
			continue
		}

		// Staged changes
		if x != ' ' && x != '?' {
			counts.Staged++
		}

		// Unstaged changes
		if y != ' ' && y != '?' {
			counts.Unstaged++
		}
	}

end:
	return counts, err
}

// StatusCountsInPathExcluding returns counts of staged, unstaged, and untracked files within a specific path,
// excluding files in specified subpaths (e.g., to exclude submodules)
func (r *Repo) StatusCountsInPathExcluding(relPath dt.PathSegments, excludePaths []dt.PathSegments) (counts StatusCounts, err error) {
	var out string
	var lines []string
	var pathPrefix string

	// Get status for the entire path
	out, err = r.runGit(context.Background(), r.Root, "status", "--porcelain", string(relPath))
	if err != nil {
		goto end
	}

	// Build path prefix for filtering (with trailing slash if not root)
	// Note: "." means current directory (same as empty string for our purposes)
	if relPath != "" && relPath != "." {
		pathPrefix = string(relPath) + "/"
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		var filePath string
		var shouldExclude bool

		if line == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}

		// Extract file path from git status output (format: "XY path" where XY is 2-char status)
		filePath = strings.TrimSpace(line[3:])

		// Remove the path prefix to get relative path within module
		if pathPrefix != "" {
			if !strings.HasPrefix(filePath, pathPrefix) {
				continue // Not in our path
			}
			filePath = strings.TrimPrefix(filePath, pathPrefix)
		}

		// Check if this file is in an excluded subpath
		shouldExclude = false
		for _, excludePath := range excludePaths {
			excludePrefix := string(excludePath) + "/"
			if strings.HasPrefix(filePath, excludePrefix) || filePath == string(excludePath) {
				shouldExclude = true
				break
			}
		}

		if shouldExclude {
			continue
		}

		// Count this file
		x := line[0] // Index (staged) status
		y := line[1] // Worktree (unstaged) status

		// Untracked files
		if x == '?' && y == '?' {
			counts.Untracked++
			continue
		}

		// Staged changes
		if x != ' ' && x != '?' {
			counts.Staged++
		}

		// Unstaged changes
		if y != ' ' && y != '?' {
			counts.Unstaged++
		}
	}

end:
	return counts, err
}

// FetchTags fetches all tags from the remote repository
func (r *Repo) FetchTags(ctx context.Context) error {
	var remoteName string

	// Get remote name (typically "origin")
	if r.Remote.Name != "" {
		remoteName = string(r.Remote.Name)
	} else {
		remoteName = "origin"
	}

	_, err := r.runGit(ctx, r.Root, "fetch", "--tags", remoteName)
	return err
}

func (r *Repo) toIntPtr(s string) (n *int) {
	n = new(int)
	*n, _ = strconv.Atoi(s)
	return n
}

func (r *Repo) Tags(ctx context.Context, prefix string) (tags []string, err error) {
	out, err := r.runGit(ctx, r.Root, "tag", "--list")
	if err != nil {
		goto end
	}

	// Normalize prefix: "." means root (same as empty)
	if prefix == "." {
		prefix = ""
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		tag := strings.TrimSpace(line)
		switch {
		case tag == "":
			continue
		case prefix == "" && strings.Contains(tag, "/"):
			continue
		case prefix != "" && !strings.HasPrefix(tag, prefix+"/"):
			continue
		}
		tags = append(tags, tag)
	}
end:
	return tags, err
}

// RemoteTags returns all tags from the remote repository
func (r *Repo) RemoteTags(ctx context.Context, prefix string) (tags []string, err error) {
	var out string
	var remoteName string

	// Get remote name (typically "origin")
	if r.Remote.Name != "" {
		remoteName = string(r.Remote.Name)
	} else {
		remoteName = "origin"
	}

	out, err = r.runGit(ctx, r.Root, "ls-remote", "--tags", remoteName)
	if err != nil {
		goto end
	}

	// Normalize prefix: "." means root (same as empty)
	if prefix == "." {
		prefix = ""
	}

	// Parse ls-remote output: "hash refs/tags/tagname" or "hash refs/tags/tagname^{}"
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ref := fields[1]
		// Skip dereferenced tags (those ending with ^{})
		if strings.HasSuffix(ref, "^{}") {
			continue
		}

		// Extract tag name from refs/tags/tagname
		if !strings.HasPrefix(ref, "refs/tags/") {
			continue
		}
		tag := strings.TrimPrefix(ref, "refs/tags/")

		// Apply prefix filter
		switch {
		case prefix == "" && strings.Contains(tag, "/"):
			continue
		case prefix != "" && !strings.HasPrefix(tag, prefix+"/"):
			continue
		}

		tags = append(tags, tag)
	}

end:
	return tags, err
}

// CompareRemoteTags checks if remote has newer tags than local
// Returns: missingTags (tags on remote but not local), error
func (r *Repo) CompareRemoteTags(ctx context.Context, prefix string) (missingTags []string, err error) {
	var localTags []string
	var remoteTags []string
	var localTagSet map[string]bool

	localTags, err = r.Tags(ctx, prefix)
	if err != nil {
		goto end
	}

	remoteTags, err = r.RemoteTags(ctx, prefix)
	if err != nil {
		goto end
	}

	// Build set of local tags for fast lookup
	localTagSet = make(map[string]bool, len(localTags))
	for _, tag := range localTags {
		localTagSet[tag] = true
	}

	// Find tags that exist on remote but not locally
	for _, tag := range remoteTags {
		if !localTagSet[tag] {
			missingTags = append(missingTags, tag)
		}
	}

	// Sort missing tags (newest first if they're semver)
	if len(missingTags) > 0 {
		sort.Slice(missingTags, func(i, j int) bool {
			return semver.Compare(missingTags[i], missingTags[j]) > 0
		})
	}

end:
	return missingTags, err
}

type LatestTagArgs struct {
	IncludeUnreachable bool // default: false
	AllowNonSemver     bool // default: false
	ModuleRelPath      dt.RelDirPath
}

// LatestTag returns the latest reachable semver tag by default.
// Passing non-default options is currently unsupported.
func (r *Repo) LatestTag(ctx context.Context, headCommit string, args *LatestTagArgs) (latest string, err error) {
	var semverTags []string
	var reachable []string
	var tags []string

	switch {
	case args == nil:
		args = &LatestTagArgs{}
	case args.IncludeUnreachable:
		panic("IncludeUnreachable option for Repo.LatestTag() is not yet implemented")
	case args.AllowNonSemver:
		panic("AllowNonSemver option for Repo.LatestTag() is not yet implemented")
	}

	tags, err = r.Tags(ctx, string(args.ModuleRelPath))

	if err != nil {
		goto end
	}

	for _, tag := range tags {
		if !semver.IsValid(tag) {
			continue
		}
		semverTags = append(semverTags, tag)
	}

	if len(semverTags) == 0 {
		err = ErrNoSemverTags
		goto end
	}

	for _, tag := range semverTags {
		ok, err := r.isAncestor(r.Root, tag, headCommit)
		if err != nil {
			goto end
		}
		if ok {
			reachable = append(reachable, tag)
		}
	}
	if len(reachable) == 0 {
		err = ErrNoReachableSemverTags
		goto end
	}

	sort.Slice(reachable, func(i, j int) bool {
		return semver.Compare(reachable[i], reachable[j]) > 0
	})
	latest = reachable[0]
end:
	return latest, err
}

func (r *Repo) isAncestor(gitDir dt.DirPath, olderRef, newerRef string) (isAncestor bool, err error) {
	var ok bool
	var ee *exec.ExitError

	cmd := exec.Command("git", "merge-base", "--is-ancestor", olderRef, newerRef)
	cmd.Dir = string(gitDir)
	err = cmd.Run()
	if err == nil {
		isAncestor = true
		goto end
	}

	ok = errors.As(err, &ee)
	if !ok {
		err = fmt.Errorf("git merge-base failed: %w", err)
		goto end
	}
	if ee.ExitCode() != 1 {
		err = fmt.Errorf("git merge-base failed with exit code %d: %w", ee.ExitCode(), err)
		goto end
	}

end:
	return isAncestor, err
}

func (r *Repo) DiffNameStatus(ctx context.Context, fromRef, toRef string, patterns ...string) (newLines []string, err error) {
	var lines []string
	var out string
	var n int
	args := []string{"diff", "--name-status", fmt.Sprintf("%s..%s", fromRef, toRef), "--"}
	args = append(args, patterns...)
	out, err = r.runGit(ctx, r.Root, args...)
	if err != nil {
		goto end
	}
	lines = strings.Split(strings.TrimSpace(out), "\n")
	newLines = make([]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		newLines[n] = line
		n++
	}
	lines = newLines[:n]
end:
	return lines, err
}

func (r *Repo) DiffNumStat(fromRef, toRef string, patterns ...string) ([]string, error) {
	args := []string{"diff", "--numstat", fmt.Sprintf("%s..%s", fromRef, toRef), "--"}
	args = append(args, patterns...)
	out, err := r.runGit(context.Background(), r.Root, args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

type CachedWorktree struct {
	Dir     dt.DirPath
	repoDir dt.DirPath
	release func() error
}

func (cw *CachedWorktree) Close() (err error) {
	if cw == nil {
		goto end
	}
	if cw.release == nil {
		goto end
	}
	err = cw.release()
	cw.release = nil
end:
	return err
}

func (cw *CachedWorktree) Checkout(ref string) error {
	if cw == nil {
		return fmt.Errorf("nil cached worktree")
	}
	if err := ensureSafeCacheRepoDir(cw.repoDir); err != nil {
		return err
	}

	if _, err := runGit(context.Background(), cw.repoDir, "checkout", "--force", ref); err != nil {
		return err
	}
	if _, err := runGit(context.Background(), cw.repoDir, "reset", "--hard", ref); err != nil {
		return err
	}
	if _, err := runGit(context.Background(), cw.repoDir, "clean", "-fdx"); err != nil {
		return err
	}
	return nil
}

func (r *Repo) OpenCachedWorktree(ctx context.Context) (wt *CachedWorktree, err error) {
	var cacheBase dt.DirPath
	var release releaseFunc

	cacheBase, err = cacheBaseDir()
	if err != nil {
		return nil, err
	}

	reposDir := dt.DirPathJoin(cacheBase, "repos")
	repoDir := dt.DirPathJoin(reposDir, repoCacheKey(r.Root))
	lockDir := dt.DirPathJoin3(cacheBase, "locks", repoCacheKey(r.Root)+".lock")

	err = ensureSameFilesystem(r.Root, cacheBase)
	if err != nil {
		goto end
	}

	release, err = acquireLock(lockDir)
	if err != nil {
		goto end
	}

	wt = &CachedWorktree{
		Dir:     repoDir,
		repoDir: repoDir,
		release: release,
	}

	if err := reposDir.MkdirAll(0o755); err != nil {
		err = joinErrs(wt.Close(), err)
		goto end
	}

	_, err = dt.FilepathJoin(repoDir, ".git").Stat()
	if err != nil {
		if !os.IsNotExist(err) {
			err = joinErrs(wt.Close(), err)
			goto end
		}
		err := cloneLocalRepo(ctx, r.Root, repoDir)
		if err != nil {
			err = joinErrs(wt.Close(), err)
			goto end
		}
	}

	err = ensureSafeCacheRepoDir(repoDir)
	if err != nil {
		err = joinErrs(wt.Close(), err)
		goto end
	}

	err = refreshFromSource(ctx, repoDir, r.Root)
	if err != nil {
		err = joinErrs(wt.Close(), err)
		goto end
	}

	_, err = runGit(ctx, repoDir, "clean", "-fdx")
	if err != nil {
		err = joinErrs(wt.Close(), err)
		goto end
	}

end:
	return wt, err
}

func joinErrs(errs ...error) (err error) {
	joined := make([]error, len(errs))
	n := 0
	for _, err = range errs {
		if err == nil {
			continue
		}
		joined[n] = err
		n++
	}
	return CombineErrs(joined)
}

func cloneLocalRepo(ctx context.Context, sourceRepoRoot, destDir dt.DirPath) (err error) {
	err = destDir.Dir().MkdirAll(0o755)
	if err != nil {
		goto end
	}
	_, err = runGit(ctx,
		"", "clone", "--local", "--no-checkout",
		string(sourceRepoRoot),
		string(destDir),
	)
end:
	return err
}

func refreshFromSource(ctx context.Context, cacheRepoDir, sourceRepoRoot dt.DirPath) error {
	// Always refresh from the source repo, ensuring we have up-to-date commits and tags.
	// This fetches branches into Remote-tracking refs to avoid updating any checked-out local branch.
	_, err := runGit(ctx, cacheRepoDir,
		"fetch", "--prune", "--tags",
		string(sourceRepoRoot),
		"+refs/heads/*:refs/remotes/source/*",
		"+HEAD:refs/remotes/source/HEAD",
	)
	return err
}

func cacheBaseDir() (baseDir dt.DirPath, err error) {
	v := strings.TrimSpace(os.Getenv("NEXTVER_CACHE_DIR"))
	if v != "" {
		baseDir, err = dt.ParseDirPath(v)
		goto end
	}
	baseDir, err = dt.UserCacheDir()
	if err != nil {
		goto end
	}
end:
	return baseDir, err
}

// TODO: Can we not just use the module path?!?
func repoCacheKey(repoRoot dt.DirPath) string {
	base := repoRoot.Clean().Base()
	sum := sha256.Sum256([]byte(repoRoot))
	return fmt.Sprintf("%s-%s", sanitize(base), hex.EncodeToString(sum[:8]))
}

func sanitize(ps dt.PathSegment) string {
	s := strings.TrimSpace(string(ps))
	if s == "" {
		return "repo"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

type releaseFunc func() error

func acquireLock(lockDir dt.DirPath) (fn releaseFunc, err error) {
	err = lockDir.Dir().MkdirAll(0o755)
	if err != nil {
		goto end
	}
	err = lockDir.MkdirAll(0o700)
	if err != nil {
		if os.IsExist(err) {
			err = fmt.Errorf("another nextver run is in progress (lock exists at %s)", lockDir)
		}
		goto end
	}
	fn = func() error { return lockDir.Remove() }
end:
	return fn, err
}

func ensureSafeCacheRepoDir(repoDir dt.DirPath) (err error) {
	var baseDir dt.DirPath
	var exists bool

	if repoDir == "" {
		err = fmt.Errorf("empty cache repo dir")
		goto end
	}
	baseDir, err = cacheBaseDir()
	if err != nil {
		goto end
	}
	baseDir, err = baseDir.Abs()
	if err != nil {
		goto end
	}
	repoDir, err = repoDir.Abs()
	if err != nil {
		goto end
	}
	if repoDir != baseDir && !repoDir.HasPrefix(baseDir.EnsureTrailSep()) {
		err = fmt.Errorf("refusing to run in non-cache directory: %s", repoDir)
		goto end
	}
	exists, err = dt.FilepathJoin(repoDir, ".git").Exists()
	if exists {
		err = fmt.Errorf("cache repo missing .git directory: %s", repoDir)
		goto end
	}
	if err != nil {
		err = fmt.Errorf("filesystem error accessing %s", repoDir)
		goto end
	}
end:
	return err
}

func (r *Repo) runGit(ctx context.Context, dir dt.DirPath, args ...string) (string, error) {
	return runGit(ctx, dir, args...)
}

func runGit(ctx context.Context, dir dt.DirPath, args ...string) (_ string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = string(dir)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			err = fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, msg)
		}
		err = fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), err
}
