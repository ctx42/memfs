## v0.5.0 (Thu, 01 Oct 2026 18:18:14 UTC)
- build(deps): bump ctx42/testing to v0.56.0 and testkit to v0.15.0.
- fix(memfs): keep ReadFrom data intact when writing mid-file.
- fix(memfs): make directory listing repeatable.
- fix(memfs): resolve nested paths and "." in fsDir.Stat.
- fix(memfs): read whole files regardless of the offset in ReadFile.
- fix(memfs): handle an offset past the end after Seek.
- fix(memfs): reject negative offsets in ReadAt and WriteAt.
- fix(memfs): stop ReadAt and WriteAt from changing the offset.
- fix(memfs): split io/fs paths on "/" in open.
- fix(memfs): walk before returning the listing in list.
- fix(memfs): reject directory cycles in AddFile.
- fix(memfs)!: return ErrOutOfBounds from FileWith instead of panicking.
- docs(memfs): correct godoc and tidy code style.
- docs: rework README and add an fs.FS example.
- style(memfs): fix gocritic and staticcheck findings in tests.
- fix(memfs): report the full path in fsDir not-found errors.

## v0.4.0 (Mon, 08 Jun 2026 06:43:39 UTC)
- chore: update dependencies and replace iokit import with testkit.
- fix(memfs): correct Seek, Truncate error type, and godoc in File.
- refactor(memfs): use min builtin in ReadDir bound calculation.
- docs: add gmdoceg markers and fix NewDirectory example in README.
- docs: add AGENTS.md with codebase guidance for AI agents.
- doc: improve code documentation.
- perf(memfs): cache sort state and add writeByte fast path.
- chore: update editor config, expand gitignore, fix trailing whitespace.

## v0.3.0 (Fri, 01 May 2026 20:07:25 UTC)
- chore: Update to Go 1.26 and update dependencies.

## v0.2.1 (Thu, 30 Apr 2026 16:30:56 UTC)
- chore: update dependencies.

## v0.2.0 (Sun, 10 Aug 2025 14:47:29 UTC)
- feat: Add memfs.NewBuffer constructor function.

## v0.1.1 (Fri, 08 Aug 2025 14:05:03 UTC)
- fix: Fix benchmark tests importing wrong memfs package.

## v0.1.0 (Thu, 07 Aug 2025 08:27:05 UTC)
- Initial implementation.

