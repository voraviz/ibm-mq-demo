# Security

## SBOM & vulnerability scanning

[`scan-sbom.sh`](scan-sbom.sh) (repo root) generates a CycloneDX SBOM with
[syft](https://github.com/anchore/syft), then scans it with **both**
[grype](https://github.com/anchore/grype) and
[trivy](https://github.com/aquasecurity/trivy). One SBOM feeds both scanners.

```bash
./scan-sbom.sh                              # default: quay.io/voravitl/simple-mq-app:latest
./scan-sbom.sh dir:api-app-go               # a Go app's source tree
./scan-sbom.sh dir:.                        # the whole repo
./scan-sbom.sh <image>:<tag>                # any built image
```

Output lands in `sbom-out/`, named by image + tag + arch (registry path
stripped); `dir:` targets have no arch suffix. Per run: the SBOM, each tool's
raw JSON, and a **consolidated** JSON:

| File | Contents |
|---|---|
| `<name>.cdx.json` | CycloneDX SBOM (syft) |
| `<name>.grype.json` | raw grype findings |
| `<name>.trivy.json` | raw trivy findings |
| `<name>.consolidated.json` | merged findings with a `source` field |
| `<name>.consolidated.csv` | same, as CSV (`source,severity,cve,package`) |

e.g. `simple-mq-app_otel_amd64.consolidated.json`, `api-app-go.consolidated.json`.

### Consolidated output

The script joins both tools' findings on `(CVE id, package)` and tags each with
a `source` field:

| `source` | Meaning |
|---|---|
| `both` | reported by grype **and** trivy (intersection) |
| `only:grype` | grype found it, trivy did not |
| `only:trivy` | trivy found it, grype did not |

It also prints a per-`source` count summary and a table to the terminal. Each
consolidated entry is `{id, pkg, severity, source}`.

Requires `syft`, `grype`, and `trivy` on `PATH`.

### Multi-architecture images

The script scans **one platform per run**. A multi-arch tag is a manifest
*list*, so the script defaults to `linux/amd64` (via syft's `--platform`) rather
than letting syft pick the host arch — that keeps results stable whether you run
it on Apple Silicon or amd64 CI. The other architectures are not cataloged.

This matters because each arch has different base-image layers → different OS
packages → potentially different CVEs. (The Java/Go dependencies are identical
across arches; the base layer is not.)

Override the arch with the `PLATFORM` env var; run once per arch to cover a
multi-arch image:

```bash
PLATFORM=linux/amd64 ./scan-sbom.sh <image>:<tag>   # default
PLATFORM=linux/arm64 ./scan-sbom.sh <image>:<tag>
```

`PLATFORM` is ignored for `dir:` targets (source trees have no architecture).

### Why run both scanners

grype and trivy pull from **overlapping but not identical** vulnerability
databases, and they suppress distro `won't-fix` CVEs differently. Neither is a
superset of the other, so **union** their reports rather than intersecting:

- A CVE found by only one tool is **not** automatically a false positive — the
  other's DB may not have ingested it, or may have suppressed it by policy.
- A CVE found by **both** is **not** automatically real — both consume NVD/GHSA,
  so a bad upstream record (wrong version range, missed distro backport) becomes
  a *shared* false positive in both.

Set membership is an input to triage, not a verdict. Confirm a finding by
checking reachability, the actual fixed version, and distro backports for that
specific CVE.
