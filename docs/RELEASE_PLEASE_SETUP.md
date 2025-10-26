# Release Please Setup Guide

This document explains how to configure the repository to allow the `release-please` workflow to function correctly.

## The Issue

The `release-please` GitHub Actions workflow automates version management and changelog generation. However, it requires specific repository permissions to create pull requests.

If you encounter this error:
```
GitHub Actions is not permitted to create or approve pull requests
```

This means the repository setting to allow GitHub Actions to create PRs is not enabled.

## Solution

### Repository Settings

You need to enable the setting that allows GitHub Actions to create and approve pull requests:

1. Go to your repository on GitHub
2. Click on **Settings**
3. In the left sidebar, click on **Actions** > **General**
4. Scroll down to the **Workflow permissions** section
5. Check the box: **"Allow GitHub Actions to create and approve pull requests"**
6. Click **Save**

### Direct Link

For this repository, you can access the settings directly at:
```
https://github.com/dbehnke/dmr-nexus/settings/actions
```

### Organization or Enterprise Settings

**Note:** If the checkbox is grayed out and you cannot change it:

- The setting may be controlled at the **organization level** (for organization repositories)
- Or at the **enterprise level** (for enterprise repositories)
- You'll need to contact your organization or enterprise admin to enable this setting

#### For Organization Admins
1. Go to: `https://github.com/organizations/YOUR_ORG/settings/actions`
2. Enable "Allow GitHub Actions to create and approve pull requests"
3. This will cascade down to all repositories in the organization

#### For Enterprise Admins
1. Go to: `https://github.com/enterprises/YOUR_ENTERPRISE/settings/actions`
2. Enable the setting at the enterprise level
3. Organizations and repositories will then be able to enable it

## Workflow Permissions

The `release-please.yml` workflow already has the correct permissions configured:

```yaml
permissions:
  contents: write
  pull-requests: write
```

These permissions are necessary but not sufficient - the repository setting must also be enabled.

## Security Context

GitHub implemented this security control in May 2022 to prevent:
- Compromised workflows from auto-merging code without review
- Unintended bypass of branch protection rules
- Automated changes without human oversight

This is a deliberate security feature, and you should only enable it if you trust the workflows in your repository.

## Verification

After enabling the setting:

1. Push a commit to the `main` branch with a conventional commit message:
   ```
   feat: add new feature
   ```
   or
   ```
   fix: resolve bug
   ```

2. The `release-please` workflow should run and create a pull request automatically

3. Check the Actions tab to verify the workflow completes successfully

## References

- [GitHub Docs: Managing GitHub Actions settings for a repository](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository)
- [GitHub Blog: Prevent GitHub Actions from creating and approving PRs](https://github.blog/changelog/2022-05-03-github-actions-prevent-github-actions-from-creating-and-approving-pull-requests/)
- [Release Please Documentation](https://github.com/googleapis/release-please)

## Troubleshooting

### Still Getting Permission Errors?

1. **Check your role**: You must be a repository admin to change these settings
2. **Check organization settings**: If in an organization, the setting may be locked at the org level
3. **Verify the setting was saved**: Sometimes changes don't persist - check the setting again
4. **Clear Actions cache**: Try re-running the workflow after enabling the setting

### Need More Help?

Open an issue in this repository and include:
- The exact error message from the workflow run
- Whether the repository is personal, organization, or enterprise
- Whether the checkbox is grayed out or available to toggle
