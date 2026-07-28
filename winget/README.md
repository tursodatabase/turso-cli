# WinGet package templates for Turso.CLI

Seed manifests for the **first** submission to [`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs).

After the package exists, do **not** hand-edit these for every release — the
[Publish to WinGet](../.github/workflows/winget.yml) workflow uses
[`wingetcreate update`](https://github.com/microsoft/winget-create/blob/main/doc/update.md).

Maintainer instructions: [docs/winget.md](../docs/winget.md)

## Before opening the first PR

1. Ship a Turso CLI GitHub Release that includes:
   - `turso-cli_Windows_x86_64.zip`
   - `turso-cli_Windows_arm64.zip`
2. Copy `manifests/0.0.0-template` to a real version directory name (for example `1.0.31`).
3. Replace `PackageVersion`, `InstallerUrl`, `InstallerSha256`, and `ReleaseDate`.
4. Compute hashes:

   ```powershell
   Get-FileHash .\turso-cli_Windows_x86_64.zip -Algorithm SHA256
   Get-FileHash .\turso-cli_Windows_arm64.zip -Algorithm SHA256
   ```

5. Validate:

   ```powershell
   winget validate --manifest .\manifests\<version>
   ```

6. Submit under `manifests/t/Turso/CLI/<version>/` in `winget-pkgs`
   (<https://learn.microsoft.com/windows/package-manager/package/repository>).

Or skip the templates and run `wingetcreate new <x64-url> <arm64-url>` instead.
