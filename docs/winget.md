## Publishing Turso CLI to WinGet

WinGet packages live in the community repository [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs).
After the package exists once, this repo’s [Publish to WinGet](../.github/workflows/winget.yml) workflow updates it on every stable GitHub Release.

Package id: `Turso.CLI`

```powershell
winget install Turso.CLI
```

### Official documentation

- Create a package manifest: <https://learn.microsoft.com/windows/package-manager/package/manifest>
- Submit the manifest (first-time registration): <https://learn.microsoft.com/windows/package-manager/package/repository>
- Authoring rules / directory layout: <https://github.com/microsoft/winget-pkgs/blob/master/doc/Authoring.md>
- WingetCreate (create / update / submit): <https://github.com/microsoft/winget-create>
- WingetCreate in CI/CD: <https://github.com/microsoft/winget-create#using-windows-package-manager-manifest-creator-in-a-cicd-pipeline>
- GitHub token for CI (use `WINGET_CREATE_GITHUB_TOKEN`, not `--token` on the CLI): <https://aka.ms/winget-create-token>
- Update command reference: <https://github.com/microsoft/winget-create/blob/main/doc/update.md>
- Policies: <https://learn.microsoft.com/windows/package-manager/package/windows-package-manager-policies>

### One-time setup (maintainers)

Do this once after a release that includes native Windows assets
(`turso-cli_Windows_x86_64.zip` and `turso-cli_Windows_arm64.zip` from GoReleaser).

1. **Create the first manifests** (interactive is fine):

   ```powershell
   winget install wingetcreate
   wingetcreate new `
     https://github.com/tursodatabase/turso-cli/releases/download/vX.Y.Z/turso-cli_Windows_x86_64.zip `
     https://github.com/tursodatabase/turso-cli/releases/download/vX.Y.Z/turso-cli_Windows_arm64.zip
   ```

   Use publisher `Turso`, package name `CLI` so the id is `Turso.CLI`.
   Installer type should be `zip` with nested portable `turso.exe` (see `winget/` for a template).

2. **Validate and open the first PR** against `microsoft/winget-pkgs` (or let WingetCreate submit):

   ```powershell
   winget validate --manifest .\manifests\t\Turso\CLI\<version>
   ```

   Follow the submit guide: <https://learn.microsoft.com/windows/package-manager/package/repository>

3. **Create a classic GitHub PAT** with the `public_repo` scope (fine-grained tokens are not supported by WingetCreate). Optional: `delete_repo` so failed forks can be cleaned up.
   Steps: <https://github.com/microsoft/winget-create/blob/main/doc/token.md>

4. **Add a repository secret** on `tursodatabase/turso-cli`:

   - Name: `WINGET_TOKEN`
   - Value: the classic PAT from step 3

   GitHub docs: <https://docs.github.com/en/actions/security-guides/encrypted-secrets#creating-encrypted-secrets-for-a-repository>

5. **Merge the first `winget-pkgs` PR**. After it lands, every later stable release can be automated.

### Continuous deployment

On each stable `release` (`published`, non-prerelease), `.github/workflows/winget.yml`:

1. Resolves `turso-cli_Windows_x86_64.zip` and `turso-cli_Windows_arm64.zip` from the release
2. Downloads WingetCreate from <https://aka.ms/wingetcreate/latest>
3. Runs `wingetcreate update Turso.CLI --version <ver> --urls <x64> <arm64> --submit`

The workflow also supports `workflow_dispatch` for a dry run (`submit: false`) or a manual submit.

Reference implementations from the WingetCreate README:

- <https://github.com/microsoft/edit/blob/main/.github/workflows/winget.yml>
- <https://github.com/microsoft/terminal/blob/main/.github/workflows/winget.yml>

### Local smoke test (optional)

```powershell
winget settings --enable LocalManifestFiles
winget install --manifest .\winget\manifests\0.0.0-template
```

Sandbox testing from a `winget-pkgs` checkout: <https://learn.microsoft.com/windows/package-manager/package/repository#step-2-test-your-manifest-with-windows-sandbox>
