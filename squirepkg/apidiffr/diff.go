package apidiffr

import (
	"errors"
	"fmt"
	"go/types"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
	"golang.org/x/exp/apidiff"
	"golang.org/x/tools/go/packages"
)

type CompareOptions struct {
	ExcludeInternalPackages bool
}

type PackagesMap map[PackagePath]*types.Package

type PackagePath = dt.DirPath

type LoadedModule struct {
	Packages PackagesMap
	Error    error
}

func DiffDirs(oldDir, newDir dt.DirPath, opts CompareOptions) (r Report, err error) {
	var newMod LoadedModule
	oldMod, err := LoadModule(oldDir)
	if err != nil {
		goto end
	}
	newMod, err = LoadModule(newDir)
	if err != nil {
		goto end
	}
	r = Diff(oldMod, newMod, opts)
end:
	return r, err
}

func Diff(oldMod, newMod LoadedModule, opts CompareOptions) Report {
	report := Report{
		LoadErrors: CombineErrs([]error{oldMod.Error, newMod.Error}),
	}

	pkgPaths := dtx.NewOrderedMap[PackagePath, struct{}](len(oldMod.Packages) + len(newMod.Packages))
	for path := range oldMod.Packages {
		pkgPaths.Set(path, struct{}{})
	}
	for path := range newMod.Packages {
		pkgPaths.Set(path, struct{}{})
	}

	paths := make([]PackagePath, 0, pkgPaths.Len())
	for path := range pkgPaths.Keys() {
		if opts.ExcludeInternalPackages && isInternalImport(path) {
			continue
		}
		paths = append(paths, path)
	}
	//slices.Sort(paths)

	for _, path := range paths {
		oldPkg := oldMod.Packages[path]
		newPkg := newMod.Packages[path]

		pc := PackageChanges{ImportPath: path}
		switch {
		case oldPkg == nil && newPkg != nil:
			pc.NonBreaking = append(pc.NonBreaking, "package added")
		case oldPkg != nil && newPkg == nil:
			pc.Breaking = append(pc.Breaking, "package removed")
		case oldPkg == nil:
			continue
		default:
			diffReport := apidiff.Changes(oldPkg, newPkg)
			for _, ch := range diffReport.Changes {
				msg := decorateChange(oldPkg, newPkg, ch.Message)
				if msg == "" {
					continue
				}
				if !ch.Compatible {
					pc.Breaking = append(pc.Breaking, msg)
				} else {
					pc.NonBreaking = append(pc.NonBreaking, msg)
				}
			}
		}

		if len(pc.Breaking) == 0 && len(pc.NonBreaking) == 0 && len(pc.Informational) == 0 {
			continue
		}

		report.Packages = append(report.Packages, pc)
	}
	return report
}

func decorateChange(oldPkg, newPkg *types.Package, msg string) string {
	target, rest, ok := strings.Cut(msg, ":")
	if !ok {
		return msg
	}
	target = strings.TrimSpace(target)
	targetRaw := target
	rest = strings.TrimSpace(rest)
	if target == "" {
		return msg
	}

	kind, display := classifyTarget(oldPkg, newPkg, target)
	if kind != "" {
		target = kind + " " + display
	} else {
		target = display
	}

	// Work around apidiff false positives where it reports a member as removed
	// even though it exists unchanged in both old and new packages.
	if rest == "removed" && strings.Contains(targetRaw, ".") {
		if fixed, handled := reconcileRemoved(oldPkg, newPkg, targetRaw); handled {
			if fixed == "" {
				return ""
			}
			rest = fixed
		}
	}

	if strings.Contains(rest, "changed from ") && strings.Contains(rest, " to ") {
		if from, to, ok := splitFromTo(rest); ok && strings.TrimSpace(from) == strings.TrimSpace(to) {
			extra := explainSameFromTo(oldPkg, newPkg, display)
			// If we can't add any clarifying detail, suppress the redundant "X -> X"
			// message (apidiff often also emits specific member changes separately).
			if extra == "" && (kind == "type" || kind == "interface" || kind == "method" || kind == "func") {
				return ""
			}
			rest = rest + " " + extra
		}
	}

	return target + ": " + rest
}

func reconcileRemoved(oldPkg, newPkg *types.Package, display string) (string, bool) {
	// display looks like Type.Member or pkg-level things; only handle Type.Member here.
	parts := strings.Split(display, ".")
	if len(parts) != 2 {
		return "", false
	}
	typeName, member := parts[0], parts[1]

	oldObj, _ := lookupMember(oldPkg, typeName, member)
	newObj, _ := lookupMember(newPkg, typeName, member)
	if oldObj == nil || newObj == nil {
		return "", false
	}

	oldSig := types.TypeString(oldObj.Type(), types.RelativeTo(oldPkg))
	newSig := types.TypeString(newObj.Type(), types.RelativeTo(newPkg))
	if oldSig != "" && newSig != "" && oldSig == newSig {
		// Member exists with identical type; drop it as a false positive.
		return "", true
	}

	if oldSig != "" && newSig != "" && oldSig != newSig {
		return fmt.Sprintf("changed from %s to %s", oldSig, newSig), true
	}

	return "", false
}

func splitFromTo(rest string) (from, to string, ok bool) {
	_, tail, ok := strings.Cut(rest, "changed from ")
	if !ok {
		return "", "", false
	}
	from, to, ok = strings.Cut(tail, " to ")
	return from, to, ok
}

func explainSameFromTo(oldPkg, newPkg *types.Package, display string) string {
	// If the message renders the same before/after (e.g. "func() T -> func() T"),
	// it often means a named type (T) changed underneath.
	// We attempt to find the first identifier that looks like a type name and compare its underlying forms.
	typeName := display
	if dot := strings.Index(typeName, "."); dot >= 0 {
		typeName = typeName[:dot]
	} else {
		for _, tok := range strings.Fields(display) {
			typeName = strings.Trim(tok, "()*[],")
			if typeName != "" {
				break
			}
		}
	}
	if typeName == "" {
		return ""
	}

	oldObj := lookupInScope(oldPkg, typeName)
	newObj := lookupInScope(newPkg, typeName)
	oldTN, _ := oldObj.(*types.TypeName)
	newTN, _ := newObj.(*types.TypeName)
	if oldTN == nil || newTN == nil {
		return ""
	}

	oldU := types.TypeString(oldTN.Type().Underlying(), types.RelativeTo(oldPkg))
	newU := types.TypeString(newTN.Type().Underlying(), types.RelativeTo(newPkg))
	if oldU == "" || newU == "" {
		return ""
	}
	if oldU != newU {
		return fmt.Sprintf("(underlying type changed from %s to %s)", oldU, newU)
	}

	if oldTN.IsAlias() != newTN.IsAlias() {
		if oldTN.IsAlias() {
			return "(changed from type alias to defined type)"
		}
		return "(changed from defined type to type alias)"
	}

	oldT := types.TypeString(oldTN.Type(), types.RelativeTo(oldPkg))
	newT := types.TypeString(newTN.Type(), types.RelativeTo(newPkg))
	if oldT != "" && newT != "" && oldT != newT {
		return fmt.Sprintf("(type declaration changed from %s to %s)", oldT, newT)
	}

	return ""
}

func classifyTarget(oldPkg, newPkg *types.Package, target string) (kind, display string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", ""
	}

	// Package-level changes.
	if strings.HasPrefix(target, "package ") {
		return "", target
	}

	parts := strings.Split(target, ".")
	switch len(parts) {
	case 1:
		name := parts[0]
		obj := lookupInScope(oldPkg, name)
		if obj == nil {
			obj = lookupInScope(newPkg, name)
		}
		if obj == nil {
			return "", target
		}
		switch o := obj.(type) {
		case *types.TypeName:
			if _, ok := o.Type().Underlying().(*types.Interface); ok {
				return "interface", name
			}
			return "type", name
		case *types.Func:
			return "func", name + "()"
		case *types.Const:
			return "const", name
		case *types.Var:
			return "var", name
		default:
			return "", target
		}
	case 2:
		typeName, member := parts[0], parts[1]
		kind, display = classifyMember(oldPkg, newPkg, typeName, member)
		if kind == "" {
			return "", target
		}
		return kind, display
	default:
		return "", target
	}
}

func classifyMember(oldPkg, newPkg *types.Package, typeName, member string) (kind, display string) {
	if typeName == "" || member == "" {
		return "", ""
	}

	if obj, isMethod := lookupMember(oldPkg, typeName, member); obj != nil {
		if isMethod {
			return "method", typeName + "." + member + "()"
		}
		if _, ok := obj.(*types.Var); ok {
			return "field", typeName + "." + member
		}
		return "", ""
	}
	if obj, isMethod := lookupMember(newPkg, typeName, member); obj != nil {
		if isMethod {
			return "method", typeName + "." + member + "()"
		}
		if _, ok := obj.(*types.Var); ok {
			return "field", typeName + "." + member
		}
		return "", ""
	}
	return "", ""
}

func lookupMember(pkg *types.Package, typeName, member string) (obj types.Object, isMethod bool) {
	if pkg == nil {
		return nil, false
	}
	tn, _ := lookupInScope(pkg, typeName).(*types.TypeName)
	if tn == nil {
		return nil, false
	}
	typ := tn.Type()
	// Prefer pointer receiver methods too.
	obj, _, _ = types.LookupFieldOrMethod(types.NewPointer(typ), true, pkg, member)
	if obj == nil {
		obj, _, _ = types.LookupFieldOrMethod(typ, true, pkg, member)
	}
	if obj == nil {
		// Fallbacks: LookupFieldOrMethod occasionally misses methods/fields for
		// some named types; search the method set and underlying struct directly.
		if named, ok := typ.(*types.Named); ok {
			for i := 0; i < named.NumMethods(); i++ {
				m := named.Method(i)
				if m != nil && m.Name() == member {
					return m, true
				}
			}
			if u, ok := named.Underlying().(*types.Struct); ok {
				for i := 0; i < u.NumFields(); i++ {
					f := u.Field(i)
					if f != nil && f.Name() == member {
						return f, false
					}
				}
			}
		}
		return nil, false
	}
	_, isMethod = obj.(*types.Func)
	return obj, isMethod
}

func lookupInScope(pkg *types.Package, name string) types.Object {
	if pkg == nil || pkg.Scope() == nil {
		return nil
	}
	return pkg.Scope().Lookup(name)
}

func LoadModule(dir dt.DirPath) (lm LoadedModule, err error) {
	var pkgs PackagesMap
	pkgs, err = loadPackages(dir)
	if errors.Is(err, ErrLoadingPackages) {
		goto end
	}
	lm = LoadedModule{
		Packages: pkgs,
		Error:    err,
	}
end:
	return lm, err
}

var ErrOnPackageLoad = errors.New("error on package load")
var ErrMissingTypeInfo = errors.New("missing type information")
var ErrLoadingPackages = errors.New("error loading packages")

func loadPackages(dir dt.DirPath) (out PackagesMap, err error) {
	var errs []error
	var pkgs []*packages.Package

	env := append(os.Environ(), "GOWORK=off")
	env = withGoFlags(env, "-mod=readonly")

	cfg := &packages.Config{
		Dir: string(dir),
		Mode: packages.NeedName |
			packages.NeedModule |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedImports,
		Env: env,
	}

	pkgs, err = packages.Load(cfg, "./...")
	if err != nil {
		err = NewErr(ErrLoadingPackages, err)
		goto end
	}

	out = make(PackagesMap)
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		if len(pkg.Errors) > 0 {
			for _, pe := range pkg.Errors {
				if isIgnorableLoadError(pe.Msg) {
					continue
				}
				errs = append(errs, NewErr(ErrOnPackageLoad,
					"package_path", pkg.PkgPath,
					fmt.Errorf("%s: %s at %s", errorKindName(pe.Kind), pe.Msg, pe.Pos)))
			}
		}
		if pkg.Types == nil {
			errs = append(errs, NewErr(ErrMissingTypeInfo,
				"package_path", pkg.PkgPath,
			))
			continue
		}
		out[PackagePath(pkg.PkgPath)] = pkg.Types
	}
	err = CombineErrs(errs)
end:
	return out, err
}

func errorKindName(kind packages.ErrorKind) string {
	switch kind {
	case packages.ListError:
		return "list"
	case packages.ParseError:
		return "parse"
	case packages.TypeError:
		return "type"
	default:
	}
	return "unknown"
}

func withGoFlags(env []string, required string) []string {
	for i := range env {
		if !strings.HasPrefix(env[i], "GOFLAGS=") {
			continue
		}
		current := strings.TrimPrefix(env[i], "GOFLAGS=")
		if strings.Contains(current, required) {
			return env
		}
		current = strings.TrimSpace(current)
		if current == "" {
			env[i] = "GOFLAGS=" + required
			return env
		}
		env[i] = "GOFLAGS=" + current + " " + required
		return env
	}
	return append(env, "GOFLAGS="+required)
}

func isIgnorableLoadError(msg string) bool {
	// Some Go toolchains emit noisy cache errors while still providing complete
	// type information. These are not actionable for API breakage reporting.
	if strings.Contains(msg, "loading compiled Go files from cache") &&
		strings.Contains(msg, "cache entry not found") {
		return true
	}
	return false
}

func isInternalImport(pkgPath PackagePath) bool {
	pkgPath = pkgPath.ToSlash()
	if pkgPath.Contains("/internal/") {
		return true
	}
	return pkgPath.HasSuffix("/internal")
}
