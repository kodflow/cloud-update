# CRITICAL: No Claude Co-Author Rule

## ⚠️ MANDATORY RULE FOR ALL AGENTS ⚠️

**NEVER add Claude as a co-author in git commits under ANY circumstances**

### Strict Git Commit Rules

When creating ANY git commit:

1. ✅ **ALWAYS** use the configured git user (check with `git config user.name` and `git config user.email`)
2. ✅ **ALWAYS** create clean commit messages WITHOUT any co-author attribution
3. ❌ **NEVER** add `Co-Authored-By: Claude` or any variant
4. ❌ **NEVER** include `Co-Authored-By: Claude <noreply@anthropic.com>`
5. ❌ **NEVER** add any Claude-related signatures, footers, or attributions
6. ❌ **NEVER** use commit trailers that reference Claude in any way

### Examples

✅ **CORRECT** - Clean commit without co-author:

```bash
git commit -m "fix: resolve authentication issue in webhook handler"
```

❌ **INCORRECT** - Never do this:

```bash
git commit -m "fix: resolve authentication issue

Co-Authored-By: Claude <noreply@anthropic.com>"
```

❌ **INCORRECT** - Also never do this:

```bash
git commit -m "fix: resolve issue

🤖 Generated with Claude
Co-Authored-By: Claude <noreply@anthropic.com>"
```

### Why This Rule Exists

- The repository owner wants all commits to be properly attributed to them
- Claude is a tool, not a contributor
- Git history should reflect human authorship only
- Professional repositories don't include AI tool attributions in commits

### Enforcement

This rule applies to:

- ALL git commits
- ALL branches
- ALL types of changes (features, fixes, chores, etc.)
- ALL agents and contexts

**NO EXCEPTIONS**
